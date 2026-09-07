set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

default:
	@just --list

web-build:
	cd web && pnpm install --frozen-lockfile && pnpm run build

web-dev:
	cd web && pnpm run dev

build: web-build
	go build -o bin/audit-log ./cmd/server

run: build
	#!/usr/bin/env bash
	set -euo pipefail
	if command -v podman >/dev/null 2>&1; then
		run_compose() { podman compose "$@"; }
	else
		run_compose() { docker compose "$@"; }
	fi
	trap 'run_compose down' EXIT
	run_compose up -d --wait
	./bin/audit-log

dev:
	#!/usr/bin/env bash
	set -euo pipefail
	trap 'kill 0; docker compose down' EXIT INT TERM
	docker compose up -d --wait
	go run ./cmd/server &
	pnpm --dir web run dev
	wait

wire:
	cd cmd/server/wire && go generate

mock:
	go generate ./internal/auditlog/usecases/...

ci:
	#!/usr/bin/env bash
	just arch
	just lint
	just unit
	just functional-api

arch:
	go run github.com/arch-go/arch-go/v2@v2.1.2

# Run golangci-lint via mise when pinned — Go 1.26+ ships golangci-lint v1 on PATH ahead of the v2 pin.
[private]
_golangci-lint +ARGS:
	#!/usr/bin/env bash
	set -euo pipefail
	if command -v mise >/dev/null 2>&1 && [[ -f .mise.toml ]]; then
		mise x golangci-lint -- golangci-lint {{ARGS}}
	else
		golangci-lint {{ARGS}}
	fi

# Depends on build: internal/web embeds dist, so the SPA must exist before the linter type-checks.
lint: build
	just _golangci-lint run ./...

unit *args:
	#!/usr/bin/env bash
	set -euo pipefail
	go test ./... -race -count=1 -coverprofile=coverage.out {{args}}
	go test ./internal/auditlog/domain ./internal/auditlog/usecases -race -count=1 -coverprofile=coverage-gate.out {{args}}
	total="$(go tool cover -func=coverage-gate.out | awk '/^total:/{gsub(/%/,"",$NF); print $NF; exit}')"
	awk -v t="${total}" 'BEGIN{if (t+0 < 60) {printf "domain+usecases coverage %.1f%% is below required 60%%\n", t+0 > "/dev/stderr"; exit 1}}'
	echo "domain+usecases coverage: ${total}% (min 60%)"

# Extra args are forwarded to `go test`, e.g.
# `just functional-api -run TestFunctional/Query_events_with_filters`.
#
# Godog (BDD) suite in test/functional, driving the service over its public gRPC API.
functional-api *args:
	#!/usr/bin/env bash
	set -euo pipefail
	just build
	export AUDIT_LOG_SERVER_PORT="${AUDIT_LOG_SERVER_PORT:-50051}"
	export AUDIT_LOG_FUNCTIONAL_GRPC_ADDR="${AUDIT_LOG_FUNCTIONAL_GRPC_ADDR:-127.0.0.1:${AUDIT_LOG_SERVER_PORT}}"
	export AUDIT_LOG_OTEL_ENABLED="${AUDIT_LOG_OTEL_ENABLED:-false}"
	cleanup() {
		if [ -n "${APP_PID:-}" ]; then
			kill -TERM "${APP_PID}" 2>/dev/null || true
			wait "${APP_PID}" 2>/dev/null || true
		fi
	}
	trap cleanup EXIT
	./bin/audit-log &
	APP_PID=$!
	host="${AUDIT_LOG_FUNCTIONAL_GRPC_ADDR%:*}"
	port="${AUDIT_LOG_FUNCTIONAL_GRPC_ADDR##*:}"
	for _ in $(seq 1 60); do
		if nc -z "${host}" "${port}" 2>/dev/null; then
			break
		fi
		sleep 1
	done
	if ! nc -z "${host}" "${port}" 2>/dev/null; then
		echo "timeout waiting for gRPC on ${host}:${port}" >&2
		exit 1
	fi
	set +e
	go test ./test/functional/... -count=1 -v {{args}}
	TEST_EXIT=$?
	set -e
	exit "${TEST_EXIT}"

# Every functional suite. Mirrors control-plane's `functional-api` / `functional-web` split.
functional *args:
	just functional-api {{args}}
	just functional-web

# Stub so tooling shared with repos that do have a browser suite (control-plane)
# can call `just functional-web` here without failing. The audit-log SPA is a
# read-only query view with no functional coverage of its own.
#
# Not applicable here: `just functional-api` is the whole functional suite.
functional-web:
	@echo "not applicable: audit-log has no web functional suite; run 'just functional-api' (gRPC) instead"

release version:
    #!/bin/bash
    set -e

    CHANGELOG_FILE_NAME="CHANGELOG.md"
    echo "📝 Updating $CHANGELOG_FILE_NAME for {{version}}..."

    git cliff -o

    echo "✅ $CHANGELOG_FILE_NAME updated"

    echo "📦 Committing changelog..."
    git add "$CHANGELOG_FILE_NAME"
    git commit -m "chore: update changelog for {{version}}"

    echo "🚀 Pushing changes..."
    git push

    echo "🏷️  Creating tag {{version}}..."
    git tag {{version}}
    git push --tags

    echo "✅ Release {{version}} complete!"

proto:
	protoc -I=proto/auditlogv1 -I="$(brew --prefix protobuf)/include" \
		--go_out=. --go_opt=module=audit-log \
		--go-grpc_out=. --go-grpc_opt=module=audit-log \
		proto/auditlogv1/audit_log.proto
