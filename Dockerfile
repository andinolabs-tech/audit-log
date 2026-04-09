FROM golang:1.26-alpine AS build
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=0.0.0-dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X 'audit-log/internal/version.Version=${VERSION}'" -o /out/audit-log ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/audit-log /audit-log
EXPOSE 50051 6061
ENTRYPOINT ["/audit-log"]
