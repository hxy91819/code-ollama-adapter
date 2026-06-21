# Code Ollama Adapter

Local Ollama Cloud proxy for Claude Code and Codex. It adapts GLM-5.2 for
`max` reasoning, 1M-class context aliases, Anthropic-compatible requests, and
OpenAI Responses API traffic.

The adapter is intentionally small: it forwards OpenAI-compatible and
Anthropic-compatible HTTP traffic to a local Ollama server, rewriting only the
metadata that these clients cannot express correctly.

## What It Adapts

- Claude Code can select `glm-5.2:cloud[1m]` so its local context budget is
  1M-class. The adapter forwards the valid Ollama model name `glm-5.2:cloud`.
- Codex can expose `xhigh` as the user-selectable top reasoning level. The
  adapter forwards Ollama GLM-compatible `max`.
- Normal `high`, `medium`, `low`, and `none` reasoning levels are left alone.

## Runtime Shape

```text
Claude Code ─┐
             ├─ http://127.0.0.1:11435 ── http://127.0.0.1:11434 ── Ollama Cloud
Codex ───────┘
```

Default rewrites:

- `model: "glm-5.2"` -> `model: "glm-5.2:cloud"`
- `model: "glm-5.2:cloud[1m]"` -> `model: "glm-5.2:cloud"`
- `reasoning.effort: "xhigh"` -> `reasoning.effort: "max"` on Responses API
- `reasoning_effort: "xhigh"` -> `reasoning_effort: "max"` on Chat Completions

The service logs method, path, status, and mutation names only. It does not log
prompts or response bodies.

## Privacy And Security

- The default service binds to `127.0.0.1`; do not expose it publicly without a
  separate authentication and network security plan.
- Do not commit API keys, provider tokens, `.env` files, local Claude/Codex
  credential files, prompt logs, response logs, or session transcripts.
- `.gitignore` excludes common local secrets, logs, build output, and temporary
  files.
- The optional pre-commit setup includes staged `gitleaks` scanning.

## Layout

- `cmd/code-ollama-adapter`: Go binary entrypoint.
- `internal/proxy`: request forwarding and payload rewrite logic.
- `systemd/code-ollama-adapter.service`: Linux systemd unit template.
- `launchd/ai.openclaw.code-ollama-adapter.plist`: macOS launchd template.
- `scripts/install.sh`: build and install for Linux/macOS.
- `scripts/uninstall.sh`: stop and remove service metadata.
- `config/codex-ollama-cloud.config.toml`: Codex profile example.
- `.agents/skills/configure-code-ollama-adapter`: Agent workflow for safely
  configuring and verifying this local setup.
- `.agents/skills/setup-codex-claude-ollama`: Agent workflow for helping users
  configure Codex and Claude Code from scratch.

## Agent Skill

This repo includes a small Codex skill for agent-assisted setup:
`setup-codex-claude-ollama`.

Ask an agent to use `$setup-codex-claude-ollama` when you want it to configure
Codex and Claude Code on a machine, preserve existing settings, back up config
files, set Codex `xhigh`, set Claude's 1M-class context alias, start the
adapter, and verify both clients.

Use `$configure-code-ollama-adapter` for narrower project maintenance and
repair after the adapter is already installed. Use the README for human
maintenance.

## Build And Test

```bash
git clone git@github.com:hxy91819/code-ollama-adapter.git
cd code-ollama-adapter
go test ./...
go build -o bin/code-ollama-adapter ./cmd/code-ollama-adapter
```

CI runs on Ubuntu and macOS and checks:

- `gofmt`
- `go test ./...`
- binary build
- install/uninstall script help output

Optional local hygiene:

```bash
pre-commit install
pre-commit run --all-files
```

The pre-commit setup expects `gitleaks` and `golangci-lint` to be installed. It
runs staged secret scanning, basic file hygiene checks, `gofmt`, `go vet`,
`go test`, and a small Go lint set.

## Run Manually

```bash
~/code-ollama-adapter/bin/code-ollama-adapter \
  --host 127.0.0.1 \
  --port 11435 \
  --upstream http://127.0.0.1:11434 \
  --model-alias glm-5.2 \
  --model-alias 'glm-5.2:cloud[1m]' \
  --model-target glm-5.2:cloud \
  --reasoning-map xhigh=max \
  --default-reasoning-effort max
```

Health check:

```bash
curl http://127.0.0.1:11435/health
```

## Install As A Service

Linux systemd:

```bash
cd ~/code-ollama-adapter
scripts/install.sh
systemctl status code-ollama-adapter.service
```

The installer builds `bin/code-ollama-adapter`, installs a root-owned executable
under `/usr/local/lib/code-ollama-adapter/<service-name>/`, and writes a systemd
unit that executes that installed binary. The service does not execute a
mutable checkout binary, and distinct service names get distinct binary paths.

macOS launchd:

```bash
cd ~/code-ollama-adapter
scripts/install.sh
launchctl print "gui/$(id -u)/ai.openclaw.code-ollama-adapter"
```

When run as root on macOS, the installer writes to
`/Library/LaunchDaemons`; otherwise it writes to `~/Library/LaunchAgents`.
The launchd plist is generated with the installed binary path.

## Uninstall

Linux:

```bash
cd ~/code-ollama-adapter
scripts/uninstall.sh
```

macOS:

```bash
cd ~/code-ollama-adapter
scripts/uninstall.sh
```

Uninstall removes service metadata only. It does not delete the project tree or
the installed binary.

## Codex Profile

`codexo` uses:

```bash
codex --profile ollama-cloud --dangerously-bypass-approvals-and-sandbox
```

The profile should point the Ollama Cloud provider at the adapter:

```toml
model = "glm-5.2"
model_provider = "ollama_cloud"
model_reasoning_effort = "xhigh"
model_catalog_json = "/absolute/path/to/ollama-cloud-glm-5.2.json"

[model_providers.ollama_cloud]
name = "Ollama Cloud via Code Ollama Adapter"
base_url = "http://127.0.0.1:11435/v1"
wire_api = "responses"
```

The model catalog should include `xhigh` in `supported_reasoning_levels`. This
setup defaults Codex to `xhigh`, and the service also injects `max` when Codex
omits reasoning from the wire request. Use `high` explicitly when you want
normal high reasoning, and `xhigh` for Ollama GLM `max`:

```bash
codex exec --profile ollama-cloud \
  -c model_reasoning_effort='"xhigh"' \
  --dangerously-bypass-approvals-and-sandbox \
  'Reply exactly OK.'
```

## Claude Code Setup

Claude Code should use the adapter as its Anthropic-compatible base URL and a
display model with the `[1m]` suffix:

```bash
ANTHROPIC_BASE_URL=http://127.0.0.1:11435
ANTHROPIC_DEFAULT_SONNET_MODEL='glm-5.2:cloud[1m]'
ANTHROPIC_DEFAULT_HAIKU_MODEL='glm-5.2:cloud[1m]'
ANTHROPIC_DEFAULT_OPUS_MODEL='glm-5.2:cloud[1m]'
```

The adapter strips `[1m]` before forwarding to Ollama. Claude Code still budgets
the session as a 1M-class model.

## Verification

Request-level smoke:

```bash
curl -sS http://127.0.0.1:11435/v1/responses \
  -H 'content-type: application/json' \
  -d '{"model":"glm-5.2","input":"Reply exactly OK.","max_output_tokens":16,"reasoning":{"effort":"xhigh"}}' \
  | jq '{model, reasoning}'
```

Expected result includes:

```json
{"reasoning":{"effort":"max"}}
```

Client smoke:

```bash
claude -p /context --model glm-5.2:cloud

codex exec --profile ollama-cloud \
  -c model_reasoning_effort='"xhigh"' \
  --dangerously-bypass-approvals-and-sandbox \
  'Reply exactly OK.'
```

Expected log markers:

- `rewrite_model`: a client alias was mapped to `glm-5.2:cloud`.
- `rewrite_reasoning`: `xhigh` was mapped to `max`.

## Maintenance

- Keep rewrite rules explicit and narrow.
- Add a test in `internal/proxy/transform_test.go` for every new rewrite rule.
- Keep the service listening on `127.0.0.1` unless there is a concrete need to
  expose it.
- Do not log request or response bodies.
- After editing behavior, run `go test ./...`, reinstall/restart the service,
  and run at least one live smoke through Codex or Claude Code.

## Migration Notes

Earlier prototypes used these names:

- project directory: `ollama-cloud-code-proxy`
- service: `ollama-cloud-code-proxy.service`
- binary: `ollama-cloud-code-proxy`
- older service: `claude-ollama-proxy.service`

The current canonical name is `code-ollama-adapter`. Keep only one service
active on port `11435`.
