## [unreleased]

### 🚜 Refactor

- Use value object within domain layer

### 🧪 Testing

- Trigger ci pipeline
- Trigger ci pipeline

### ⚙️ Miscellaneous Tasks

- Remove comments
## [0.1.3] - 2026-04-10

### 🚀 Features

- Add file and shared connection when sqlite

### ⚙️ Miscellaneous Tasks

- Update changelog for v0.1.3
## [0.1.2] - 2026-04-10

### 🐛 Bug Fixes

- In-memory implementation use a library that support static compilation

### ⚙️ Miscellaneous Tasks

- Update changelog for v0.1.2
## [0.1.1] - 2026-04-10

### ⚙️ Miscellaneous Tasks

- Add the ghcr server for the final docker image
- Update changelog for v0.1.1
## [0.1.0] - 2026-04-09

### 🚀 Features

- First working version
- Make small docker images available to pull and run
- Add mandatory fields validations
- Make AUDIT_LOG_DB_DSN optional in config
- Add database.OpenInMemory with SQLite :memory:
- Fall back to SQLite in-memory DB when AUDIT_LOG_DB_DSN is unset

### 🐛 Bug Fixes

- Remove create role from bootstrap
- *(auditlog)* Align namespace and occurred_at across gRPC, use cases, and DB

### 📚 Documentation

- Update run section of README.md
- Add tech assessment analysis with many improvements
- Add in-memory implementation design
- Update readme

### ⚙️ Miscellaneous Tasks

- Add drone as ci tool
- Change just installer
- Change drone publing to push docker image
- Support only amd64 with drone
- Set version dynamically in compilation time
- Swtich from drone to woodpecker
- Remove postgres for functional test given they are not longer require
- Remove comment
- Udpate github registry token
- Add just release command
- Update changelog for v0.1.0
