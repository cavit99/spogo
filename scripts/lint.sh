#!/usr/bin/env bash
set -euo pipefail

go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2 run "$@"
