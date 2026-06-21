# AGENTS.md

Prioritize single responsibility for functions and files.

This project is a local integration shim for Codex, Claude Code, and Ollama Cloud.
Keep it boring: explicit request rewrites, no prompt logging, no unrelated client behavior.

Decision comments should explain why a rewrite exists, not what the Go syntax does.

Before changing deployed behavior:

- Add or update a focused test in `internal/proxy/transform_test.go`.
- Run `go test ./...`.
- Restart the systemd service and run one live smoke request.
