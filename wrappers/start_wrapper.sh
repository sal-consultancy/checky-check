#!/usr/bin/env bash
set -euo pipefail

# Usage: ./wrappers/start_wrapper.sh [config-file-or-directory] [--check]
# Defaults: config-sample, port 8070. Override with CHECKYCHECK_CONFIG_PATH / CHECKYCHECK_PORT.
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${CHECKYCHECK_CONFIG_PATH:-$repo_root/config-sample}"
run_checks=false
config_given=false
for arg in "$@"; do
  case "$arg" in
    --check) run_checks=true ;;
    -h|--help)
      echo "Usage: $0 [config-file-or-directory] [--check]"
      echo "Builds the frontend and serves the app. --check runs configured checks before serving."
      echo "Set config secrets in the environment (config-sample requires cc_passphrase)."
      exit 0 ;;
    -*) echo "Unknown option: $arg" >&2; exit 1 ;;
    *)
      if "$config_given"; then echo "Only one config path is supported." >&2; exit 1; fi
      config_path="$arg"
      config_given=true ;;
  esac
done
if [[ "$config_path" != /* ]]; then config_path="$PWD/$config_path"; fi
if [[ ! -e "$config_path" ]]; then echo "Config not found: $config_path" >&2; exit 1; fi
cd "$repo_root"
if [[ ! -d frontend/node_modules ]]; then npm --prefix frontend ci; fi
npm --prefix frontend run build
if "$run_checks"; then go run . -mode=check -config="$config_path"; fi
echo "Starting local app at http://localhost:${CHECKYCHECK_PORT:-8070} using $config_path"
exec go run . -mode=serve -config="$config_path" -port="${CHECKYCHECK_PORT:-8070}"
