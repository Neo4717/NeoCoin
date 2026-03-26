#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TARGETS=(
  blockchain/data
  blockchain/data-node0
  blockchain/data-node1
  blockchain/data-node2
)

echo "Fixing ownership/permissions for local chain data dirs..."
echo "Targets:"
printf ' - %s\n' "${TARGETS[@]}"
echo

if command -v sudo >/dev/null 2>&1; then
  if sudo -n true >/dev/null 2>&1; then
    sudo chown -R "$(id -u)":"$(id -g)" "${TARGETS[@]}"
    sudo chmod -R u+rwX,go+rX "${TARGETS[@]}"
    echo "OK"
    exit 0
  fi

  cat <<'EOF'
sudo needs a password in this environment.

Re-run interactively:
  sudo ./scripts/fix_data_perms.sh
EOF
  exit 1
fi

echo "sudo not found; cannot fix root-owned files."
exit 1
