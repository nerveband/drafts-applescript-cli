#!/bin/bash
set -euo pipefail
go test -mod=readonly ./...
goreleaser release --snapshot --clean
