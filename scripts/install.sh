#!/usr/bin/env bash
set -euo pipefail

# Definition:
#   Build and install the Codex/Claude Ollama Cloud adapter as a Linux systemd
#   service or macOS launchd job.
# Parameters:
#   --service-name: Linux systemd unit name. Default: code-ollama-adapter.service.
#   --label: macOS launchd label. Default: ai.openclaw.code-ollama-adapter.
#   --no-restart: install files without starting/restarting.
# Outputs:
#   Rebuilds bin/code-ollama-adapter, installs the executable, and installs
#   platform service metadata.
# Examples:
#   scripts/install.sh
#   scripts/install.sh --no-restart

usage() {
  cat <<'EOF'
Usage:
  scripts/install.sh [options]

Description:
  Build and install Code Ollama Adapter as a Linux systemd service or macOS
  launchd job.

Options:
  --service-name <name>  Linux systemd unit name. Default: code-ollama-adapter.service
  --label <label>        macOS launchd label. Default: ai.openclaw.code-ollama-adapter
  --no-restart           Install without starting/restarting.
  -h, --help             Show this help.

Outputs:
  bin/code-ollama-adapter is rebuilt.
  The executable is installed into a service-specific directory under
  /usr/local/lib/code-ollama-adapter on Linux and root macOS installs, or
  ~/.local/lib/code-ollama-adapter for user macOS LaunchAgents.
  Linux: /etc/systemd/system/<service-name> is installed and enabled.
  macOS: a launchd plist is installed into ~/Library/LaunchAgents, or
         /Library/LaunchDaemons when running as root.

Examples:
  scripts/install.sh
  scripts/install.sh --service-name code-ollama-adapter.service
EOF
}

service_name="code-ollama-adapter.service"
label="ai.openclaw.code-ollama-adapter"
restart=1

while (($#)); do
  case "$1" in
    --service-name)
      [[ $# -ge 2 && -n "$2" ]] || { echo "ERROR: --service-name requires a value" >&2; exit 2; }
      service_name="$2"
      shift 2
      ;;
    --label)
      [[ $# -ge 2 && -n "$2" ]] || { echo "ERROR: --label requires a value" >&2; exit 2; }
      label="$2"
      shift 2
      ;;
    --no-restart)
      restart=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
project_root="$(cd -- "$script_dir/.." && pwd)"
os_name="$(uname -s)"

case "$os_name" in
  Linux)
    service_id="${service_name%.service}"
    if [[ ! "$service_id" =~ ^[A-Za-z0-9@_.-]+$ ]]; then
      echo "ERROR: --service-name must contain only letters, digits, @, _, ., or -" >&2
      exit 2
    fi
    install_dir="/usr/local/lib/code-ollama-adapter/$service_id"
    ;;
  Darwin)
    if [[ ! "$label" =~ ^[A-Za-z0-9_.-]+$ ]]; then
      echo "ERROR: --label must contain only letters, digits, _, ., or -" >&2
      exit 2
    fi
    if [[ "$(id -u)" == "0" ]]; then
      install_dir="/usr/local/lib/code-ollama-adapter/$label"
    else
      install_dir="$HOME/.local/lib/code-ollama-adapter/$label"
    fi
    ;;
  *)
    echo "ERROR: unsupported OS: $os_name" >&2
    exit 1
    ;;
esac

(cd "$project_root" && go build -o bin/code-ollama-adapter ./cmd/code-ollama-adapter)
chmod 0755 "$project_root/bin/code-ollama-adapter"
mkdir -p "$install_dir"
install_bin="$install_dir/code-ollama-adapter"
install -m 0755 "$project_root/bin/code-ollama-adapter" "$install_bin"
install_bin_sed="${install_bin//\\/\\\\}"
install_bin_sed="${install_bin_sed//&/\\&}"
install_bin_sed="${install_bin_sed//|/\\|}"

case "$os_name" in
  Linux)
    unit_src="$project_root/systemd/code-ollama-adapter.service"
    unit_dst="/etc/systemd/system/$service_name"
    tmp_unit="$(mktemp)"
    sed "s|__BIN_PATH__|$install_bin_sed|g" "$unit_src" > "$tmp_unit"
    install -m 0644 "$tmp_unit" "$unit_dst"
    rm -f "$tmp_unit"
    systemctl daemon-reload
    systemctl enable "$service_name" >/dev/null
    if [[ "$restart" == "1" ]]; then
      systemctl restart "$service_name"
      systemctl --no-pager --full status "$service_name" | sed -n '1,80p'
    else
      echo "Installed $unit_dst; restart skipped."
    fi
    ;;
  Darwin)
    if [[ "$(id -u)" == "0" ]]; then
      plist_dir="/Library/LaunchDaemons"
      domain="system"
    else
      plist_dir="$HOME/Library/LaunchAgents"
      domain="gui/$(id -u)"
    fi
    mkdir -p "$plist_dir"
    plist_dst="$plist_dir/$label.plist"
    sed "s#ai.openclaw.code-ollama-adapter#$label#g" \
      "$project_root/launchd/ai.openclaw.code-ollama-adapter.plist" \
      | sed "s|__BIN_PATH__|$install_bin_sed|g" > "$plist_dst"
    chmod 0644 "$plist_dst"
    if [[ "$restart" == "1" ]]; then
      launchctl bootout "$domain" "$plist_dst" >/dev/null 2>&1 || true
      launchctl bootstrap "$domain" "$plist_dst"
      launchctl kickstart -k "$domain/$label"
      launchctl print "$domain/$label" | sed -n '1,80p'
    else
      echo "Installed $plist_dst; restart skipped."
    fi
    ;;
  *)
    echo "ERROR: unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac
