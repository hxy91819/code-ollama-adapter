---
name: setup-codex-claude-ollama
description: Configure or repair a user's local Codex and Claude Code setup so both clients use Ollama Cloud GLM through code-ollama-adapter. Use when asked to set up Codex profiles, Claude Code settings, Ollama Cloud GLM-5.2, optional cheaper Claude Haiku defaults such as GLM-4.7, max reasoning, xhigh-to-max mapping, 1M-class Claude context, local adapter URLs, or end-to-end verification for these coding agents.
---

# Setup Codex Claude Ollama

Use this skill to help a user configure Codex and Claude Code for Ollama Cloud
GLM through `code-ollama-adapter`. Prioritize preserving the user's existing
configuration while adding the minimal required profile/settings.

## Guardrails

- Do not print API tokens, auth files, shell secrets, prompt logs, response
  bodies, or session transcripts.
- Back up any config file before editing it. Use timestamped backups such as
  `settings.json.bak.YYYYMMDDHHMMSS`.
- Merge settings. Do not overwrite existing Claude hooks, permissions,
  statusLine, enabledPlugins, Codex project trust entries, or user aliases.
- Keep the adapter and Ollama endpoints bound to `127.0.0.1` unless the user
  explicitly asks for remote access and accepts the security tradeoff.
- Treat Ollama GLM `max` reasoning as the desired default for coding-agent use.
- For Codex, expose `xhigh` as the selectable user-facing level and rely on the
  adapter to send Ollama-compatible `max`.
- For Claude Code, keep Sonnet/Opus on `glm-5.2:cloud[1m]` for the 1M-class
  context path, and use `glm-4.7:cloud` as the cheaper Haiku-class default when
  the user wants lower-cost lightweight tasks.
- Verify behavior with clients and adapter logs, not only by inspecting files.

## Discover Paths

Inspect before editing:

```bash
pwd
command -v codex || true
command -v claude || true
command -v ollama || true
echo "${CODEX_HOME:-}"
echo "${CLAUDE_HOME:-}"
```

Use these default locations when environment variables are unset:

- Codex home: `${CODEX_HOME:-$HOME/.codex}`
- Claude home: `${CLAUDE_HOME:-$HOME/.claude}`
- Adapter clone: current repository, or a user-selected clone of
  `git@github.com:hxy91819/code-ollama-adapter.git`

If the machine already has custom wrappers, aliases, or homes, follow those
paths instead of forcing defaults.

## Adapter Baseline

Install and start the adapter first:

```bash
cd /path/to/code-ollama-adapter
go test ./...
scripts/install.sh
systemctl is-active code-ollama-adapter.service 2>/dev/null || true
curl -sS http://127.0.0.1:11435/health
```

Expected service behavior:

- listen: `http://127.0.0.1:11435`
- upstream Ollama: `http://127.0.0.1:11434`
- model target: `glm-5.2:cloud`
- aliases: `glm-5.2`, `glm-5.2:cloud[1m]`
- reasoning map: `xhigh=max`
- default reasoning injection: `max`

## Codex Configuration

Create or update a Codex profile named `ollama-cloud`. Preserve unrelated
settings in existing TOML files.

Profile target:

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

Model catalog requirements:

- model slug: `glm-5.2`
- `default_reasoning_level`: `xhigh`
- supported reasoning includes at least `none`, `high`, and `xhigh`
- `xhigh` description should say it maps to Ollama GLM `max`
- `context_window`: `976000`
- `max_context_window`: `976000`
- `auto_compact_token_limit`: around `927000`

If Codex shows `xhigh` in the UI but sends no reasoning field, that is expected
for some versions. The adapter's `--default-reasoning-effort max` is the safety
net that still sends `max` upstream.

Optional convenience alias:

```bash
codexo() {
  codex --profile ollama-cloud --dangerously-bypass-approvals-and-sandbox "$@"
}
```

Only add aliases after checking the user's shell startup files. Do not duplicate
an existing alias.

## Claude Code Configuration

Update Claude settings by merging into `settings.json`. Keep existing keys.

Recommended values:

```json
{
  "model": "glm-5.2:cloud[1m]",
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "ollama",
    "ANTHROPIC_API_KEY": "",
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:11435",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "glm-4.7:cloud",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "glm-5.2:cloud[1m]",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "glm-5.2:cloud[1m]",
    "CLAUDE_CODE_MAX_CONTEXT_TOKENS": "1000000",
    "CLAUDE_CODE_AUTO_COMPACT_WINDOW": "950000",
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
  }
}
```

If the installed Claude Code version supports an explicit reasoning effort
setting, set it to `max`. If it does not expose that setting, do not invent
unknown keys unless the user asks; rely on the adapter and model defaults.

Claude uses `glm-5.2:cloud[1m]` only as a client-side display/context alias.
The adapter strips `[1m]` and forwards `glm-5.2:cloud` to Ollama.
Do not add `[1m]` to `glm-4.7:cloud`; it is a cheaper Haiku-class default with
a smaller context window.

## Verification

First verify the adapter rewrites requests correctly:

```bash
curl -sS http://127.0.0.1:11435/v1/responses \
  -H 'content-type: application/json' \
  -d '{"model":"glm-5.2","input":"Reply exactly OK.","max_output_tokens":16}' \
  | jq '{model, reasoning}'
```

Expected: `reasoning.effort` is `max`.

Verify that explicit lower effort is still selectable:

```bash
curl -sS http://127.0.0.1:11435/v1/responses \
  -H 'content-type: application/json' \
  -d '{"model":"glm-5.2","input":"Reply exactly OK.","max_output_tokens":16,"reasoning":{"effort":"high"}}' \
  | jq '{model, reasoning}'
```

Expected: `reasoning.effort` is `high`.

Verify Codex:

```bash
timeout 180 codex exec --profile ollama-cloud \
  --dangerously-bypass-approvals-and-sandbox \
  'Reply exactly OK.'
```

Expected: Codex reports model `glm-5.2`, provider `ollama_cloud`, reasoning
effort `xhigh`, and returns `OK`.

Verify Claude:

```bash
timeout 180 claude -p 'Reply exactly OK.' --model 'glm-5.2:cloud[1m]'
timeout 180 claude -p 'Reply exactly OK.' --model 'glm-4.7:cloud'
```

Expected: Claude returns `OK` for both. For interactive Claude, also check
`/status` and `/context` on the GLM-5.2 path; the context should be 1M-class
rather than the default 200K.

Check adapter logs without exposing bodies:

```bash
journalctl -u code-ollama-adapter.service --no-pager -n 40 \
  | rg 'listening|POST /v1/responses|POST /v1/messages|rewrite'
```

Expected markers:

- `rewrite_model` for client aliases.
- `rewrite_reasoning` when `xhigh` or omitted reasoning becomes `max`.

## Troubleshooting

- If Claude still shows 200K context, confirm it launched from a fresh terminal
  after settings changed and that `CLAUDE_CODE_MAX_CONTEXT_TOKENS` is visible in
  its environment.
- If Haiku should be cheaper but still shows GLM-5.2, confirm
  `ANTHROPIC_DEFAULT_HAIKU_MODEL` is `glm-4.7:cloud` and restart Claude Code.
- If Codex says `xhigh` but logs only `rewrite_model`, confirm the service was
  restarted with `--default-reasoning-effort max`.
- If requests fail before reaching Ollama, check `ollama` is listening on
  `127.0.0.1:11434`.
- If the adapter returns 404 for model names, confirm the client uses an alias
  handled by the adapter and the upstream target is `glm-5.2:cloud`.
- If the user has custom model names, change aliases and target together, then
  add focused tests before changing deployed behavior.

## Final Response

Summarize files changed, backups created, service state, client smoke results,
and any unresolved assumptions. Do not include secret values.
