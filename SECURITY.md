# Security

This adapter is intended to run on localhost only. Keep it bound to
`127.0.0.1` unless you have a separate authentication and network security
plan.

## Secrets

Do not commit:

- API keys or provider tokens
- `.env` files
- local Claude, Codex, Ollama, or shell credential files
- prompt logs, response logs, or session transcripts
- private keys or certificates

The adapter logs request method, path, status, and rewrite markers only. It
does not intentionally log prompt or response bodies.

## Reporting

Please open a private security advisory or contact the maintainer before
publishing a vulnerability that exposes local credentials or prompt data.
