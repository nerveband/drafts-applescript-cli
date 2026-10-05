#!/bin/bash
set -euo pipefail
if [[ "${1:-}" != "--publish" || "$#" != 1 ]]; then
  printf '%s\n' 'Publication requires explicit authorization. Usage: ./release.sh --publish' >&2
  exit 2
fi
go test -mod=readonly ./...
go vet -mod=readonly ./...
release_tag="$(git describe --tags --exact-match HEAD)"
goreleaser release --clean --skip=publish
(cd dist && shasum -a 256 -c checksums.txt)
gh release create "$release_tag" \
  --repo nerveband/drafts-applescript-cli \
  --verify-tag --title "$release_tag" --notes-file CHANGELOG.md \
  dist/checksums.txt dist/drafts_darwin_amd64.tar.gz dist/drafts_darwin_arm64.tar.gz
