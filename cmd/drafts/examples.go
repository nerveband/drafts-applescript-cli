package main

import (
	"fmt"
	assets "github.com/nerveband/drafts-applescript-cli/v4"
	"reflect"
	"strings"
)

var bundledSkill = assets.Skill

const exampleUUID = "12345678-1234-1234-1234-123456789abc"

var commandExamples = map[string]string{
	"sync":          "drafts sync --filter all --limit 100 --dry-run",
	"new":           `drafts new --input '{"content":"Meeting notes","tags":["work"]}' --dry-run`,
	"create":        `drafts create --input '{"content":"Meeting notes","tags":["work"]}' --dry-run`,
	"get":           "drafts get " + exampleUUID + " --fields uuid,title",
	"append":        `drafts append --input '{"uuid":"` + exampleUUID + `","content":"Next step"}' --dry-run`,
	"prepend":       `drafts prepend --input '{"uuid":"` + exampleUUID + `","content":"Summary"}' --dry-run`,
	"replace":       `drafts replace --input '{"uuid":"` + exampleUUID + `","content":"Revised notes"}' --dry-run`,
	"update":        `drafts update --input '{"uuid":"` + exampleUUID + `","content":"Revised notes"}' --dry-run`,
	"edit":          "drafts edit " + exampleUUID + " --dry-run",
	"select":        "drafts select",
	"list":          "drafts list --limit 5 --sort modified --created-after 2026-01-01",
	"flag":          "drafts flag " + exampleUUID + " --dry-run",
	"unflag":        "drafts unflag " + exampleUUID + " --dry-run",
	"archive":       "drafts archive " + exampleUUID + " --dry-run",
	"inbox":         "drafts inbox " + exampleUUID + " --dry-run",
	"trash":         "drafts trash " + exampleUUID + " --dry-run",
	"delete":        "drafts delete " + exampleUUID + " --dry-run",
	"open":          "drafts open " + exampleUUID + " --dry-run",
	"tag":           "drafts tag " + exampleUUID + " -t work --dry-run",
	"remove-tags":   "drafts remove-tags " + exampleUUID + " -t work --dry-run",
	"tags":          "drafts tags",
	"workspaces":    "drafts workspaces",
	"workspace":     "drafts workspace --open Work --dry-run",
	"actions":       "drafts actions --search Copy",
	"run":           `drafts run --input '{"action":"Copy","uuid":"` + exampleUUID + `"}' --dry-run`,
	"apps":          "drafts apps",
	"info":          "drafts info",
	"schema":        "drafts schema --detected",
	"agent-context": "drafts agent-context create",
	"upgrade":       "drafts upgrade --dry-run",
	"version":       "drafts version",
	"profile":       `drafts profile save work --input '{"app":"/Applications/Drafts.app","channel":"auto","timeout":"30s"}' --dry-run`,
	"config":        "drafts config",
	"feedback":      `drafts feedback "list help needs a date example" --dry-run`,
	"jobs":          "drafts jobs",
	"skills":        "drafts skills",
}

func printCommandExample(args Args) {
	v := reflect.ValueOf(args)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() == reflect.Pointer && !v.Field(i).IsNil() {
			name := strings.TrimPrefix(t.Field(i).Tag.Get("arg"), "subcommand:")
			if e, ok := commandExamples[name]; ok {
				fmt.Println("\nExample:\n  " + e)
				return
			}
		}
	}
	fmt.Println("\nExamples:\n  drafts apps\n  drafts schema\n  " + commandExamples["create"])
}
