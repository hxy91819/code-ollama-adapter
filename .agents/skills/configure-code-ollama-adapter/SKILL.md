---
name: configure-code-ollama-adapter
description: Use when configuring, auditing, repairing, or explaining this machine's Codex and Claude Code setup for GLM through Ollama Cloud via code-ollama-adapter. Covers local profiles, Claude settings, service install/restart, reasoning effort max defaults, context-window verification, and safe smoke tests.
---

# Configure Code Ollama Adapter

Use this skill when the user wants an agent to configure or verify Codex and
Claude Code against Ollama Cloud GLM through this adapter. Keep changes local,
explicit, and reversible.

## Policy

- Treat Ollama GLM `max` reasoning as the desired default for this setup.
- For Codex, expose the selectable top level as `xhigh`; the adapter maps
  `xhigh` to Ollama's `max`.
- Also keep `--default-reasoning-effort max` in the service, because Codex may
  display `xhigh` while omitting reasoning from the wire request.
- Keep the service bound to `127.0.0.1`. Do not expose it on a public interface.
- Do not print API tokens, auth files, prompts, or response bodies in the final
  answer or logs.
- Prefer local configuration changes before changing adapter code. Change code
  only when a client cannot express the needed request shape.

## Files To Inspect

- Adapter project: current repository root.
- Service template: `systemd/code-ollama-adapter.service`.
- Codex profile: `$CODEX_HOME/ollama-cloud.config.toml`,
  `~/.codex/ollama-cloud.config.toml`, or the user's configured Codex home.
- Codex model catalog: the absolute path referenced by `model_catalog_json`.
- Claude settings: `$CLAUDE_HOME/settings.json`, `~/.claude/settings.json`,
  or the user's configured Claude home.
- Shell alias: the user's shell startup file, such as `~/.bashrc` or `~/.zshrc`.

## Expected Configuration

Claude Code:

- `ANTHROPIC_BASE_URL=http://127.0.0.1:11435`
- default Sonnet/Opus models use `glm-5.2:cloud[1m]`
- default Haiku model uses `glm-4.7:cloud` when the user wants a cheaper
  lightweight model
- `effortLevel=max`
- `CLAUDE_CODE_MAX_CONTEXT_TOKENS=1000000`
- `CLAUDE_CODE_AUTO_COMPACT_WINDOW=950000`

Codex:

- profile `ollama-cloud` uses `base_url = "http://127.0.0.1:11435/v1"`
- profile uses `model = "glm-5.2"`
- profile uses `model_reasoning_effort = "xhigh"`
- model catalog includes `default_reasoning_level = "xhigh"`
- model catalog context is 976000-class, with auto compact below that limit

Adapter service:

- canonical unit is `code-ollama-adapter.service`
- listens on `127.0.0.1:11435`
- forwards to `http://127.0.0.1:11434`
- starts with `--reasoning-map xhigh=max`
- starts with `--default-reasoning-effort max`

## Workflow

1. Inspect current config and service state before editing.
2. If only local settings are wrong, update the relevant config file or service
   template; do not edit Go code.
3. If adapter behavior changes, add or update a focused test in
   `internal/proxy/transform_test.go`.
4. Run:

   ```bash
   cd /path/to/code-ollama-adapter
   gofmt -w cmd/code-ollama-adapter/main.go internal/proxy/*.go
   go test ./...
   scripts/install.sh
   ```

5. Verify the service:

   ```bash
   systemctl is-active code-ollama-adapter.service
   curl -sS http://127.0.0.1:11435/health
   ```

6. Verify default max injection:

   ```bash
   curl -sS http://127.0.0.1:11435/v1/responses \
     -H 'content-type: application/json' \
     -d '{"model":"glm-5.2","input":"Reply exactly OK.","max_output_tokens":16}' \
     | jq '{model, reasoning}'
   ```

   Expected: `reasoning.effort` is `max`.

7. Verify explicit `high` remains selectable:

   ```bash
   curl -sS http://127.0.0.1:11435/v1/responses \
     -H 'content-type: application/json' \
     -d '{"model":"glm-5.2","input":"Reply exactly OK.","max_output_tokens":16,"reasoning":{"effort":"high"}}' \
     | jq '{model, reasoning}'
   ```

   Expected: `reasoning.effort` is `high`.

8. Verify clients:

   ```bash
   timeout 180 codex exec \
     --profile ollama-cloud \
     --dangerously-bypass-approvals-and-sandbox \
     'Reply exactly OK.'

   timeout 180 claude -p 'Reply exactly OK.' \
     --model 'glm-5.2:cloud[1m]'
   timeout 180 claude -p 'Reply exactly OK.' \
     --model 'glm-4.7:cloud'
   ```

9. Check logs for mutation markers without exposing bodies:

   ```bash
   journalctl -u code-ollama-adapter.service --no-pager -n 40 \
     | rg 'listening|POST /v1/responses|POST /v1/messages|rewrite'
   ```

## Final Response

Report the effective behavior, files changed, tests run, service state, and any
remaining risk. Mention exact paths. Do not include secret values.
