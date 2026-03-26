#!/bin/bash
#
# NeoCoin Update Checker - Lightweight version check
# Usage: ./scripts/check-update.sh
#

REPO="Neo4717/NeoCoin"
CURRENT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "local")

echo "NeoCoin Update Checker"
echo "====================="
echo "Current: $CURRENT_COMMIT"

# Check GitHub for latest
LATEST=$(curl -sS "https://api.github.com/repos/${REPO}/commits/main" | grep -o '"sha": *"[^"]*"' | cut -d'"' -f4 | cut -c1-7)

if [ -z "$LATEST" ]; then
    echo "Latest:  (could not fetch)"
    echo "Status:  Unknown"
    exit 1
fi

echo "Latest:   $LATEST"

if [ "$CURRENT_COMMIT" != "$LATEST" ]; then
    echo "Status:  UPDATE AVAILABLE"
    echo ""
    echo "To update:"
    echo "  git pull origin main"
    echo "  go build -o neocoin ./cmd/node/"
    exit 1
else
    echo "Status:  UP TO DATE"
    exit 0
fi
