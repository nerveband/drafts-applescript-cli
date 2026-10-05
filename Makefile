.PHONY: build verify test native-qa
build:
	go build -mod=readonly -o drafts ./cmd/drafts

test:
	go test -mod=readonly ./...

verify: test
	go vet -mod=readonly ./...
	go build -mod=readonly ./cmd/drafts

native-qa:
	@test -n "$(DRAFTS_DICTIONARY_APP)" || (printf '%s\n' 'Set DRAFTS_DICTIONARY_APP to an installed app for dictionary-only QA' >&2; exit 2)
	go test -mod=readonly ./pkg/drafts -run '^TestNativeAppleScript$$' -count=1
