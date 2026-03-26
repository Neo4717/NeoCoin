#!/bin/bash
#
# NeoCoin Safe Updater - With Testing & Rollback
# Only updates after tests pass
#

set -e

REPO="Neo4717/NeoCoin"
LOG_FILE="${LOG_FILE:-/var/log/neocoin-updater.log}"
DATA_DIR="${DATA_DIR:-./data}"
CHECK_INTERVAL="${CHECK_INTERVAL:-7200}"
AUTO_UPDATE="${AUTO_UPDATE:-false}"
TEST_BEFORE_UPDATE="${TEST_BEFORE_UPDATE:-true}"
KEEP_BACKUPS="${KEEP_BACKUPS:-3}"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

check_github_version() {
    local current_commit="$1"
    local latest_commit
    
    latest_commit=$(curl -sS "https://api.github.com/repos/${REPO}/commits/main" 2>/dev/null | grep -o '"sha": *"[^"]*"' | cut -d'"' -f4 | cut -c1-7)
    
    if [ -z "$latest_commit" ]; then
        log "ERROR: Could not fetch latest commit"
        return 1
    fi
    
    if [ "$current_commit" != "$latest_commit" ]; then
        log "Update available: $current_commit -> $latest_commit"
        return 0
    fi
    
    log "Already up to date: $current_commit"
    return 1
}

backup_data() {
    local backup_dir="${DATA_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
    
    if [ -d "$DATA_DIR" ]; then
        log "Creating backup at $backup_dir"
        cp -r "$DATA_DIR" "$backup_dir"
        echo "$backup_dir"
        
        # Clean old backups
        local backup_count=$(ls -1d "${DATA_DIR}.backup."* 2>/dev/null | wc -l)
        if [ "$backup_count" -gt "$KEEP_BACKUPS" ]; then
            ls -1td "${DATA_DIR}.backup."* | tail -n +$((KEEP_BACKUPS + 1)) | xargs -r rm -rf
            log "Cleaned old backups"
        fi
    fi
}

test_build() {
    log "Running build test..."
    
    if ! go build -o neocoin ./cmd/node/ 2>&1 | tee -a "$LOG_FILE"; then
        log "ERROR: Build failed - NOT updating"
        return 1
    fi
    
    log "Build successful"
    return 0
}

test_run() {
    log "Running quick functional test..."
    
    # Start node in background
    timeout 10 ./neocoin server &
    local pid=$!
    
    sleep 5
    
    # Check if responding
    if curl -sf http://localhost:8080/chain/info > /dev/null 2>&1; then
        log "Functional test passed"
        kill $pid 2>/dev/null || true
        return 0
    fi
    
    log "ERROR: Functional test failed - NOT updating"
    kill $pid 2>/dev/null || true
    return 1
}

safe_update() {
    local current_commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    log "Starting safe update process..."
    log "Current commit: $current_commit"
    
    # Check for updates
    if ! check_github_version "$current_commit"; then
        log "No update available"
        return 0
    fi
    
    # Create backup
    backup_data
    local backup_dir="$?"
    
    # Fetch latest
    log "Fetching latest code..."
    git fetch origin
    
    # Stash any local changes
    git stash || true
    
    # Checkout latest
    log "Checking out latest..."
    git checkout origin/main
    
    # Test build
    if [ "$TEST_BEFORE_UPDATE" == "true" ]; then
        if ! test_build; then
            log "Build failed - rolling back"
            git checkout "$current_commit"
            return 1
        fi
        
        if ! test_run; then
            log "Test failed - rolling back"
            git checkout "$current_commit"
            return 1
        fi
    else
        log "WARNING: Skipping tests (TEST_BEFORE_UPDATE=false)"
        if ! test_build; then
            log "Build failed - rolling back"
            git checkout "$current_commit"
            return 1
        fi
    fi
    
    log "UPDATE COMPLETED SUCCESSFULLY!"
    log "New commit: $(git rev-parse --short HEAD)"
    
    # Restart node
    if [ -f "neocoin.pid" ]; then
        local pid=$(cat neocoin.pid)
        if kill -0 "$pid" 2>/dev/null; then
            log "Restarting node (PID: $pid)"
            kill -HUP "$pid" || true
        fi
    fi
}

# Main
case "${1:-check}" in
    check)
        check_github_version "$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
        ;;
    update)
        safe_update
        ;;
    run)
        log "Starting safe update loop..."
        while true; do
            safe_update
            sleep "$CHECK_INTERVAL"
        done
        ;;
esac
