#!/bin/bash
#
# NeoCoin Backup Script
# Creates backups without deleting any data

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

BACKUP_DIR="$SCRIPT_DIR/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="neocoin_backup_$TIMESTAMP"

echo "========================================"
echo "  NeoCoin - Backup Script"
echo "========================================"
echo "Creating backup: $BACKUP_NAME"

mkdir -p "$BACKUP_DIR"

echo "Backing up blockchain data..."
if [ -f "$SCRIPT_DIR/data/chain.db" ]; then
    cp -p "$SCRIPT_DIR/data/chain.db" "$BACKUP_DIR/${BACKUP_NAME}_chain.db"
    echo "  ✓ chain.db"
fi

echo "Backing up Tor keys (keeps onion address)..."
if [ -d "$SCRIPT_DIR/data/tor" ]; then
    cp -rp "$SCRIPT_DIR/data/tor" "$BACKUP_DIR/${BACKUP_NAME}_tor"
    echo "  ✓ Tor keys"
fi

echo "Backing up wallet..."
if [ -f "$SCRIPT_DIR/data/wallet.json" ]; then
    cp -p "$SCRIPT_DIR/data/wallet.json" "$BACKUP_DIR/${BACKUP_NAME}_wallet.json"
    echo "  ✓ wallet.json"
fi

echo ""
echo "Backup complete: $BACKUP_NAME"
echo "Location: $BACKUP_DIR/"
echo ""

# Keep only last 10 backups
cd "$BACKUP_DIR"
ls -t neocoin_backup_* 2>/dev/null | tail -n +11 | xargs -r rm -f
echo "Cleanup complete (keeping last 10 backups)"