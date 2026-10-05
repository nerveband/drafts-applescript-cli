#!/usr/bin/env python3
"""Explicitly authorized live checks, restricted to newly created fixture drafts."""
import json
import os
import pathlib
import pty
import shutil
import subprocess
import tempfile
import sys
import uuid

if os.environ.get("DRAFTS_LIVE_FIXTURES") != "1":
    raise SystemExit("Set DRAFTS_LIVE_FIXTURES=1 after authorizing fixture-only live tests")
app = os.environ.get("DRAFTS_TEST_APP", "")
if not pathlib.Path(app).is_absolute():
    raise SystemExit("DRAFTS_TEST_APP must be absolute; start this app explicitly first")
binary = str(pathlib.Path(os.environ.get("DRAFTS_TEST_BINARY", "./drafts")).resolve())
tag = "cli-live-" + uuid.uuid4().hex
created = []
checks = 0

state = tempfile.mkdtemp(prefix="drafts-live-")
journal = pathlib.Path(state) / "fixtures.json"
def save_journal():
    journal.write_text(json.dumps({"app": app, "tag": tag, "uuids": created}))
    journal.chmod(0o600)
save_journal()
print("Fixture recovery journal: " + str(journal), flush=True)

def run():
    def invoke(command, *args, payload=None, raw=False):
        argv = [binary, "--app", app, "--state-dir", state, command, *args]
        if payload is not None:
            argv += ["--input", "-"]
        result = subprocess.run(argv, input=json.dumps(payload).encode() if payload is not None else b"",
                                capture_output=True, timeout=45)
        if result.returncode:
            raise RuntimeError(f"{command} failed: {result.stderr.decode()}")
        return result.stdout.decode() if raw else json.loads(result.stdout)["data"]

    def check(condition, label):
        global checks
        if not condition:
            raise AssertionError(label)
        checks += 1
        print("PASS " + label, flush=True)

    def destructive(command, target, payload=None):
        args = [target] if payload is None else []
        plan = invoke(command, *args, "--dry-run", payload=payload)
        check(plan["dry_run"] is True, command + " preview")
        return invoke(command, *args, "--commit", payload=payload)

    def edit_with_helper(target, source):
        editor = pathlib.Path(state) / "conflict-editor.py"
        editor.write_text(source)
        editor.chmod(0o600)
        check(invoke("edit", target, "--dry-run")["dry_run"], "edit preview")
        master, slave = pty.openpty()
        try:
            return subprocess.run([binary, "--app", app, "edit", target, "--commit"], stdin=slave,
                                  capture_output=True, timeout=45,
                                  env={**os.environ, "EDITOR": f"'{sys.executable}' '{editor}'"})
        finally:
            os.close(master)
            os.close(slave)

    try:
        text = "CLI isolated fixture\t☃\nsecond ||| line\r\n"
        request = {"content": text, "tags": [tag, tag + "-case", (tag + "-case").upper()]}
        check(invoke("create", "--dry-run", payload=request)["dry_run"], "create preview")
        first = invoke("create", "--idempotency-key", tag + "-first", payload=request)["uuid"]
        created.append(first)
        save_journal()
        replay = invoke("create", "--idempotency-key", tag + "-first", payload=request)
        check(replay["uuid"] == first and replay["status"] == "replayed", "durable create replay")
        native_case_tags = [t for t in invoke("get", first)["tags"] if t.lower() == tag + "-case"]
        check(len(native_case_tags) == 2, "native Drafts tags preserve case")
        check(invoke("get", first)["content"] == text, "Unicode and newline round trip")
        check(invoke("get", first, "--format", "raw", raw=True) == text, "raw bytes")
        invoke("append", payload={"uuid": first, "content": "tail", "separator": ""})
        invoke("prepend", payload={"uuid": first, "content": "head", "separator": ""})
        check(invoke("get", first)["content"] == "head" + text + "tail", "append and prepend")
        destructive("replace", first, payload={"uuid": first, "content": text})
        check(invoke("get", first)["content"] == text, "replace")
        for original, concurrent in [("Case", "case"), ("Cafe\u0301", "CAF\u00c9")]:
            destructive("replace", first, payload={"uuid": first, "content": original})
            result = edit_with_helper(first, "import json,pathlib,subprocess,sys\n" +
                              "subprocess.run(" + repr([binary, "--app", app, "replace", "--input", "-", "--commit"]) +
                              ", input=json.dumps(" + repr({"uuid": first, "content": concurrent}) +
                              ").encode(), capture_output=True, check=True, timeout=45)\n" +
                              "pathlib.Path(sys.argv[-1]).write_text('Edited fixture\\n')\n")
            check(result.returncode == 6 and json.loads(result.stderr)["error"]["code"] == "CONFLICT",
                  "exact edit conflict: " + repr(original))
            check(invoke("get", first)["content"] == concurrent, "concurrent content preserved")
        destructive("replace", first, payload={"uuid": first, "content": "Cafe\u0301"})
        result = edit_with_helper(first, "import pathlib,sys\npathlib.Path(sys.argv[-1]).write_text('Edited fixture\\n')\n")
        check(result.returncode == 0 and invoke("get", first)["content"] == "Edited fixture\n",
              "unchanged decomposed Unicode permits edit")
        destructive("replace", first, payload={"uuid": first, "content": text})
        invoke("tag", payload={"uuid": first, "tags": [tag + "-extra"]})
        check(tag + "-extra" in invoke("get", first)["tags"], "tag")
        invoke("remove-tags", payload={"uuid": first, "tags": [tag + "-extra"]})
        check(tag + "-extra" not in invoke("get", first)["tags"], "remove tags")
        invoke("tag", payload={"uuid": first, "tags": [tag + "-case", (tag + "-case").upper()]})
        case_tags = [t for t in invoke("get", first)["tags"] if t.lower() == tag + "-case"]
        check(len(case_tags) == 2, "case sensitive tag addition")
        invoke("remove-tags", payload={"uuid": first, "tags": [(tag + "-case").upper()]})
        remaining = invoke("get", first)["tags"]
        check(tag + "-case" in remaining and (tag + "-case").upper() not in remaining, "case sensitive tag removal")
        check(invoke("list", "--tag", (tag + "-case").upper(), "--count")["count"] == 1,
              "native tag queries ignore case")
        invoke("flag", first)
        check(invoke("get", first)["isFlagged"], "boolean flag")
        detected = invoke("schema", "--detected")
        if detected["flag_type_available"]:
            invoke("flag", first, "--flag-type", "2")
            check(invoke("get", first)["flagType"] == 2, "optional flag type")
        invoke("unflag", first)
        check(not invoke("get", first)["isFlagged"], "unflag")
        second = invoke("create", "--idempotency-key", tag + "-second", payload={"content": "Second fixture", "tags": [tag]})["uuid"]
        created.append(second)
        save_journal()
        rows = invoke("list", "--filter", "all", "--tag", tag, "--full", "--limit", "5")
        check({d["uuid"] for d in rows["drafts"]} == set(created), "scoped list")
        check(invoke("list", "--filter", "all", "--tag", tag, "--count")["count"] == 2, "scoped count")
        check(invoke("list", "--tag", tag, "--search", "second |||")["count"] == 1, "content search")
        check(invoke("list", "--tag", tag, "--created-after", "2020-01-01")["count"] == 2, "date filter")
        failed_key = tag + "-missing-action"
        result = subprocess.run([binary, "--app", app, "--state-dir", state, "run", "--input", "-",
                                 "--idempotency-key", failed_key],
                                input=json.dumps({"uuid": first, "action": tag + "-missing"}).encode(),
                                capture_output=True, timeout=45)
        check(result.returncode == 3 and json.loads(result.stderr)["error"]["code"] == "ACTION_NOT_FOUND",
              "missing action preflight")
        check(invoke("jobs", failed_key)["jobs"] == [], "failed preflight leaves key unreserved")
        invoke("archive", first)
        check(invoke("get", first)["folder"] == "archive", "archive")
        invoke("inbox", first)
        check(invoke("get", first)["folder"] == "inbox", "restore inbox")
        invoke("sync", "--filter", "all", "--tag", tag, "--full", "--limit", "5")
        check(invoke("get", first, "--data-source", "local")["content"] == text, "snapshot get")
        check(invoke("list", "--data-source", "local", "--tag", tag)["count"] == 2, "snapshot list")
    finally:
        failures = []
        for target in created:
            try:
                destructive("trash", target)
                check(invoke("get", target)["isTrashed"], "fixture cleanup")
            except Exception as error:
                failures.append((target, str(error)))
        if failures:
            raise RuntimeError("Fixture cleanup failed: " + repr(failures))
    print(f"Passed {checks} live checks; {len(created)} fixture drafts moved to Trash", flush=True)
    shutil.rmtree(state)

run()
