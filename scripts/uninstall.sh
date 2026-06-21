#!/usr/bin/env bash
set -euo pipefail

# Definition:
#   Uninstall the Code Ollama Adapter service metadata for Linux systemd or
#   macOS launchd. The project files and built binary are left in place.
# Parameters:
#   --service-name: Linux systemd unit name. Default: code-ollama-adapter.service.
#   --label: macOS launchd label. Default: ai.openclaw.code-ollama-adapter.
# Outputs:
#   Stops/disables/removes service metadata where present.
# Examples:
#   scripts/uninstall.sh
#   scripts/uninstall.sh --service-name old-name.service

usage() {
  cat <<'EOF'
Usage:
  scripts/uninstall.sh [options]

Description:
  Stop and remove the Code Ollama Adapter Linux systemd service or macOS launchd
  job. Project files are not deleted.

Options:
  --service-name <name>  Linux systemd unit name. Default: code-ollama-adapter.service
  --label <label>        macOS launchd label. Default: ai.openclaw.code-ollama-adapter
  -h, --help             Show this help.

Outputs:
  Linux: stops/disables/removes /etc/systemd/system/<service-name>.
  macOS: bootouts/removes the installed launchd plist.

Examples:
  scripts/uninstall.sh
  scripts/uninstall.sh --label ai.openclaw.code-ollama-adapter
EOF
}

service_name="code-ollama-adapter.service"
label="ai.openclaw.code-ollama-adapter"

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

case "$(uname -s)" in
  Linux)
    service_id="${service_name%.service}"
    if [[ ! "$service_id" =~ ^[A-Za-z0-9@_.-]+$ ]]; then
      echo "ERROR: --service-name must contain only letters, digits, @, _, ., or -" >&2
      exit 2
    fi
    unit_dst="/etc/systemd/system/$service_name"
    systemctl stop "$service_name" >/dev/null 2>&1 || true
    systemctl disable "$service_name" >/dev/null 2>&1 || true
    rm -f "$unit_dst"
    systemctl daemon-reload
    systemctl reset-failed "$service_name" >/dev/null 2>&1 || true
    echo "Removed Linux service $service_name"
    ;;
  Darwin)
    if [[ ! "$label" =~ ^[A-Za-z0-9_.-]+$ ]]; then
      echo "ERROR: --label must contain only letters, digits, _, ., or -" >&2
      exit 2
    fi
    if [[ "$(id -u)" == "0" ]]; then
      plist_dir="/Library/LaunchDaemons"
      domain="system"
    else
      plist_dir="$HOME/Library/LaunchAgents"
      domain="gui/$(id -u)"
    fi
    plist_dst="$plist_dir/$label.plist"
    launchctl bootout "$domain" "$plist_dst" >/dev/null 2>&1 || true
    rm -f "$plist_dst"
    echo "Removed launchd job $label"
    ;;
  *)
    echo "ERROR: unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac
