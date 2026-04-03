set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

default:
	@just --list

build:
	go build -o bin/audit-log ./cmd/server

run: build
	#!/usr/bin/env bash
	set -euo pipefail
	trap 'docker compose down' EXIT
	docker compose up -d --wait
	./bin/audit-log

wire:
	cd cmd/server/wire && go generate

mock:
	go generate ./internal/auditlog/usecases/...

arch:
	go run github.com/arch-go/arch-go/v2@v2.1.2

unit *args:
	go test ./... -race -count=1 -coverprofile=coverage.out {{args}}

functional *args:
	#!/usr/bin/env bash
	set -euo pipefail
	just build
	if [ "${GITHUB_ACTIONS:-}" != "true" ]; then
		docker compose up -d --wait
	fi
	export AUDIT_LOG_SERVER_PORT="${AUDIT_LOG_SERVER_PORT:-50051}"
	export AUDIT_LOG_FUNCTIONAL_GRPC_ADDR="${AUDIT_LOG_FUNCTIONAL_GRPC_ADDR:-127.0.0.1:${AUDIT_LOG_SERVER_PORT}}"
	export AUDIT_LOG_DB_DSN="${AUDIT_LOG_DB_DSN:-postgres://audit:audit@localhost:5432/auditlog?sslmode=disable}"
	export AUDIT_LOG_DB_ADMIN_DSN="${AUDIT_LOG_DB_ADMIN_DSN:-postgres://audit:audit@localhost:5432/auditlog?sslmode=disable}"
	export AUDIT_LOG_OTEL_ENABLED="${AUDIT_LOG_OTEL_ENABLED:-false}"
	cleanup() {
		if [ -n "${APP_PID:-}" ]; then
			kill -TERM "${APP_PID}" 2>/dev/null || true
			wait "${APP_PID}" 2>/dev/null || true
		fi
		if [ "${GITHUB_ACTIONS:-}" != "true" ]; then
			docker compose down || true
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

release v:
	@echo "Tag release {{v}} (implement git tag/push in CI)"

proto:
	protoc -I=proto/auditlogv1 -I="$(shell brew --prefix protobuf)/include" \
		--go_out=. --go_opt=module=audit-log \
		--go-grpc_out=. --go-grpc_opt=module=audit-log \
		proto/auditlogv1/audit_log.proto
