#!/bin/bash
# NeoCoin Backup Script
# Auto-backup every 5 hours

NEO_COIN_DIR="$HOME/Download/Neocoin"
BACKUP_DIR="$HOME/neocoin-backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Check if blockchain data exists
if [ ! -f "$NEO_COIN_DIR/blockchain/data/chain.db" ]; then
    echo "$(date): ❌ No blockchain data found"
    exit 1
fi

# Copy database
cp -r "$NEO_COIN_DIR/blockchain/data/chain.db" "$BACKUP_DIR/chain_$TIMESTAMP.db"

# Keep only last 10 backups
cd "$BACKUP_DIR"
ls -t chain_*.db | tail -n +11 | xargs -r rm

echo "$(date): ✅ Backup saved: chain_$TIMESTAMP.db"

# Show backup count
BACKUP_COUNT=$(ls -1 chain_*.db 2>/dev/null | wc -l)
echo "$(date): 📦 Total backups: $BACKUP_COUNT"
