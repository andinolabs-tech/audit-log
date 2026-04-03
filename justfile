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
	go test ./test/functional/... -count=1 -v {{args}}

migrate *args:
	@echo "Run server once with AUDIT_LOG_DB_ADMIN_DSN set, or apply SQL from persistence.BootstrapSQL"

release v:
	@echo "Tag release {{v}} (implement git tag/push in CI)"

proto:
	protoc -I=proto/auditlogv1 -I="$(shell brew --prefix protobuf)/include" \
		--go_out=. --go_opt=module=audit-log \
		--go-grpc_out=. --go-grpc_opt=module=audit-log \
		proto/auditlogv1/audit_log.proto
