#!/usr/bin/env bash
set -euo pipefail

go tool gofumpt -w .
go tool gci write --skip-generated -s standard -s default -s blank -s dot .
