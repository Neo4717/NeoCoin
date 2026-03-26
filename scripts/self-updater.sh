#!/bin/bash
#
# NeoCoin Self-Updater
# Automatically checks for updates and upgrades the node
#

set -e

REPO="Neo4717/NeoCoin"
CURRENT_VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")
UPDATE_BRANCH="main"
CHECK_INTERVAL=${CHECK_INTERVAL:-3600}  # Default: 1 hour
AUTO_UPDATE=${AUTO_UPDATE:-false}
BACKUP_ENABLED=${BACKUP_ENABLED:-true}

LOG_FILE="${LOG_FILE:-/var/log/neocoin-updater.log}"
DATA_DIR="${DATA_DIR:-./data}"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "NeoCoin Self-Updater Starting..."
log "Current version: $CURRENT_VERSION"

check_for_updates() {
    log "Checking for updates..."
    
    # Get latest release info from GitHub
    RELEASE_JSON=$(curl -sS "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)
    
    if [ -z "$RELEASE_JSON" ]; then
        log "Warning: Could not fetch release info"
        return 1
    fi
    
    LATEST_VERSION=$(echo "$RELEASE_JSON" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$LATEST_VERSION" ]; then
        log "Warning: Could not parse version"
        return 1
    fi
    
    log "Latest version: $LATEST_VERSION"
    
    if [ "$LATEST_VERSION" != "$CURRENT_VERSION" ]; then
        log "Update available: $CURRENT_VERSION -> $LATEST_VERSION"
        return 0
    else
        log "Already up to date"
        return 1
    fi
}

backup_data() {
    if [ "$BACKUP_ENABLED" != "true" ]; then
        log "Backup disabled, skipping..."
        return 0
    fi
    
    if [ ! -d "$DATA_DIR" ]; then
        log "No data directory found, skipping backup"
        return 0
    fi
    
    BACKUP_DIR="${DATA_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
    log "Creating backup at $BACKUP_DIR"
    
    cp -r "$DATA_DIR" "$BACKUP_DIR"
    log "Backup created successfully"
}

perform_update() {
    log "Starting update process..."
    
    # Check for updates first
    if ! check_for_updates; then
        log "No updates available"
        return 0
    fi
    
    # Backup data if enabled
    backup_data
    
    log "Fetching latest code..."
    git fetch origin
    
    log "Pulling latest changes..."
    git checkout origin/$UPDATE_BRANCH
    
    log "Building updated binary..."
    go build -o neocoin ./cmd/node/
    
    log "Restarting node..."
    # Signal node to reload or restart
    if [ -f "neocoin.pid" ]; then
        PID=$(cat neocoin.pid)
        if kill -0 "$PID" 2>/dev/null; then
            log "Sending reload signal to node (PID: $PID)"
            kill -HUP "$PID" || true
        fi
    fi
    
    log "Update completed successfully!"
}

run_update_loop() {
    log "Starting update check loop (interval: ${CHECK_INTERVAL}s)"
    
    while true; do
        if check_for_updates; then
            if [ "$AUTO_UPDATE" == "true" ]; then
                log "Auto-update enabled, performing update..."
                perform_update
            else
                log "Auto-update disabled. Run with AUTO_UPDATE=true to enable"
            fi
        fi
        
        sleep "$CHECK_INTERVAL"
    done
}

# Main execution
case "${1:-run}" in
    check)
        check_for_updates
        ;;
    update)
        perform_update
        ;;
    run)
        run_update_loop
        ;;
    *)
        echo "Usage: $0 {check|update|run}"
        echo "  check  - Check for updates without updating"
        echo "  update - Perform update now"
        echo "  run    - Run update loop (default)"
        exit 1
        ;;
esac
