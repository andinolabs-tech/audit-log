set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

default:
	@just --list

web-build:
	cd web && npm ci && npm run build

web-dev:
	cd web && npm run dev

build: web-build
	go build -o bin/audit-log ./cmd/server

run: build
	#!/usr/bin/env bash
	set -euo pipefail
	trap 'docker compose down' EXIT
	docker compose up -d --wait
	./bin/audit-log

dev:
	#!/usr/bin/env bash
	set -euo pipefail
	trap 'kill 0; docker compose down' EXIT INT TERM
	docker compose up -d --wait
	go run ./cmd/server &
	npm --prefix web run dev
	wait

wire:
	cd cmd/server/wire && go generate

mock:
	go generate ./internal/auditlog/usecases/...

arch:
	go run github.com/arch-go/arch-go/v2@v2.1.2

unit *args:
	#!/usr/bin/env bash
	set -euo pipefail
	go test ./... -race -count=1 -coverprofile=coverage.out {{args}}
	go test ./internal/auditlog/domain ./internal/auditlog/usecases -race -count=1 -coverprofile=coverage-gate.out {{args}}
	total="$(go tool cover -func=coverage-gate.out | awk '/^total:/{gsub(/%/,"",$NF); print $NF; exit}')"
	awk -v t="${total}" 'BEGIN{if (t+0 < 60) {printf "domain+usecases coverage %.1f%% is below required 60%%\n", t+0 > "/dev/stderr"; exit 1}}'
	echo "domain+usecases coverage: ${total}% (min 60%)"

functional *args:
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
	protoc -I=proto/auditlogv1 -I="$(shell brew --prefix protobuf)/include" \
		--go_out=. --go_opt=module=audit-log \
		--go-grpc_out=. --go-grpc_opt=module=audit-log \
		proto/auditlogv1/audit_log.proto
