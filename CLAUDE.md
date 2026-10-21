# CLAUDE.md

Go service. See README.md.

- Test: `go test ./...`
- Vet: `go vet ./...`
- Build: `CGO_ENABLED=0 go build ./cmd/optimizer`
- Dependency bumps: `go get <module>@<version>` then `go mod tidy`; commit `go.mod` and `go.sum` together.
- Keep `internal/solver` free of I/O so it stays unit-testable.
- Branches `fix/<topic>`; PRs via `gh pr create`; security fixes name the CVE in the PR title.
