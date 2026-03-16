from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel
from typing import Optional, Dict, Any, Tuple
import os
import json
import hashlib
import re
from datetime import datetime
from collections import defaultdict
import threading
from advanced_ai import (
    ml_score_transaction,
    record_transaction,
    get_address_profile,
    get_global_stats,
    detect_anomaly_zscore,
    analyze_frequency,
)

# Thread-safe audit log storage
audit_lock = threading.Lock()
audit_log: list[Dict[str, Any]] = []
stats = {"total_audited": 0, "approved": 0, "rejected": 0, "errors": 0}

app = FastAPI()

# Configuration
AUDIT_LOG_PATH = os.getenv("AUDIT_LOG_PATH", "/app/data/audit.log")
AUDIT_MODE = os.getenv("AUDIT_MODE", "local")  # "local" or "llm"
LLM_API_KEY = os.getenv("LLM_API_KEY", "")
LLM_PROVIDER = os.getenv("LLM_PROVIDER", "gemini")

# Detection patterns (no external API needed)
DANGEROUS_PATTERNS = [
    # Smart contract exploitation
    r"reentrancy",
    r"overflow",
    r"underflow",
    r"selfdestruct",
    r"delegatecall",
    # Fraud patterns
    r"ponzi",
    r"pyramid",
    r"rug.?pull",
    r"scam",
    r"phish",
    r"fake.*ico",
    r"airdrop.?scam",
    # Malware/attacks
    r"exploit",
    r"hack",
    r"bruteforce",
    r"keylogger",
    r"miner",
    r"cryptojack",
    # Illegal content
    r"child.?porn",
    r"csam",
    r"terrorist",
    r"drug.*sale",
    r"weapon.*trade",
]

# Compile patterns for efficiency
compiled_dangerous = [re.compile(p, re.IGNORECASE) for p in DANGEROUS_PATTERNS]

# Suspicious patterns (lower confidence)
SUSPICIOUS_PATTERNS = [
    r"free.*token",
    r"claim.*now",
    r"double.*your",
    r"guaranteed.*return",
    r"no.*risk",
    r"urgent.*action",
    r"限.*时",
    r"免费.*代币",
    r"立即.*领取",
]

compiled_suspicious = [re.compile(p, re.IGNORECASE) for p in SUSPICIOUS_PATTERNS]

# High-risk addresses (known bad actors - can be extended)
HIGH_RISK_ADDRESSES: set = set()

# Load high-risk addresses from file if exists
RISK_LIST_PATH = os.getenv("RISK_LIST_PATH", "/app/data/high_risk_addresses.txt")
if os.path.exists(RISK_LIST_PATH):
    try:
        with open(RISK_LIST_PATH, "r") as f:
            for line in f:
                addr = line.strip()
                if addr and not addr.startswith("#"):
                    HIGH_RISK_ADDRESSES.add(addr.lower())
    except Exception:
        pass


class Transaction(BaseModel):
    sender: str
    recipient: str
    amount: int
    data: str = ""


class AIRequest(BaseModel):
    transaction: Transaction


class AIResponse(BaseModel):
    valid: bool
    reason: Optional[str] = None
    risk_score: Optional[float] = None
    checks_passed: Optional[list[str]] = None


def analyze_local(tx: Transaction) -> tuple[bool, str, float, list[str]]:
    """
    Local pattern-based analysis without external LLM.
    Returns: (is_valid, reason, risk_score, checks_passed)
    """
    checks_passed = []
    risk_factors = []
    risk_score = 0.0

    sender = tx.sender.lower() if tx.sender else ""
    recipient = tx.recipient.lower() if tx.recipient else ""
    data = tx.data.lower() if tx.data else ""
    combined = f"{sender} {recipient} {data}"

    # Check 1: High-risk address
    if sender in HIGH_RISK_ADDRESSES:
        risk_factors.append("sender_in_high_risk_list")
        risk_score += 50.0
    if recipient in HIGH_RISK_ADDRESSES:
        risk_factors.append("recipient_in_high_risk_list")
        risk_score += 50.0

    # Check 2: Zero amount (data-only transaction)
    if tx.amount == 0:
        checks_passed.append("zero_amount_checked")

    # Check 3: Very large amount (potential whale dump)
    if tx.amount > 1_000_000:
        risk_factors.append("large_amount")
        risk_score += 10.0 * (tx.amount / 1_000_000)

    # Check 4: Dangerous patterns in data
    for pattern in compiled_dangerous:
        if pattern.search(data):
            risk_factors.append(f"dangerous_pattern:{pattern.pattern}")
            risk_score += 40.0

    # Check 5: Suspicious patterns
    suspicious_count = 0
    for pattern in compiled_suspicious:
        if pattern.search(data):
            suspicious_count += 1
            risk_score += 15.0

    if suspicious_count > 0:
        risk_factors.append(f"suspicious_patterns:{suspicious_count}")

    # Check 6: Address format validation
    if sender and not re.match(r"^[a-f0-9]{64}$", sender):
        risk_factors.append("invalid_sender_format")
        risk_score += 20.0

    if recipient and not re.match(r"^[a-f0-9]{64}$", recipient):
        risk_factors.append("invalid_recipient_format")
        risk_score += 20.0

    # Check 7: Same sender and recipient
    if sender and sender == recipient:
        risk_factors.append("self_transfer")
        risk_score += 30.0

    # Check 8: Data contains only hex (potential binary/executable)
    if data and re.match(r"^[0-9a-f]+$", data) and len(data) > 1000:
        risk_factors.append("large_hex_data")
        risk_score += 25.0

    # Check 9: URL in data (potential phishing)
    if re.search(r"https?://", data):
        risk_factors.append("contains_url")
        risk_score += 15.0

    # Check 10: Unicode lookalike characters (IDN homograph attack)
    if data and any(ord(c) > 127 for c in data):
        risk_factors.append("unicode_characters")
        risk_score += 10.0

    # Determine validity based on risk score
    if risk_score >= 50.0:
        return False, f"High risk score: {risk_score:.1f}", risk_score, checks_passed
    elif risk_score >= 25.0:
        return (
            True,
            f"Medium risk score: {risk_score:.1f} - allowed with caution",
            risk_score,
            checks_passed,
        )
    else:
        return True, "Low risk - transaction approved", risk_score, checks_passed


def analyze_ml(tx: Transaction) -> Tuple[bool, str, float]:
    """
    ML-based analysis using statistical anomaly detection.
    Returns: (is_valid, reason, ml_risk_score)
    """
    sender = tx.sender.lower() if tx.sender else ""
    recipient = tx.recipient.lower() if tx.recipient else ""
    data = tx.data.lower() if tx.data else ""
    timestamp = datetime.utcnow()

    ml_score = ml_score_transaction(sender, recipient, tx.amount, data, timestamp)

    record_transaction(sender, recipient, tx.amount)

    if ml_score >= 60.0:
        return False, f"ML detected high risk: {ml_score:.1f}", ml_score
    elif ml_score >= 35.0:
        return True, f"ML moderate risk: {ml_score:.1f}", ml_score
    else:
        return True, f"ML low risk: {ml_score:.1f}", ml_score


def log_audit(request: AIRequest, response: AIResponse, process_time_ms: float):
    """Thread-safe audit logging"""
    timestamp = datetime.utcnow().isoformat() + "Z"
    tx_hash = hashlib.sha256(
        f"{request.transaction.sender}{request.transaction.recipient}{request.transaction.amount}".encode()
    ).hexdigest()[:16]

    entry = {
        "timestamp": timestamp,
        "tx_hash_prefix": tx_hash,
        "sender": request.transaction.sender[:16] + "..."
        if len(request.transaction.sender) > 16
        else request.transaction.sender,
        "recipient": request.transaction.recipient[:16] + "..."
        if len(request.transaction.recipient) > 16
        else request.transaction.recipient,
        "amount": request.transaction.amount,
        "valid": response.valid,
        "reason": response.reason,
        "risk_score": response.risk_score,
        "process_time_ms": round(process_time_ms, 2),
    }

    with audit_lock:
        audit_log.append(entry)
        # Keep only last 10000 entries in memory
        if len(audit_log) > 10000:
            audit_log[:] = audit_log[-10000:]

    # Also write to file if path is set
    try:
        os.makedirs(os.path.dirname(AUDIT_LOG_PATH), exist_ok=True)
        with open(AUDIT_LOG_PATH, "a") as f:
            f.write(json.dumps(entry) + "\n")
    except Exception:
        pass  # Don't fail if logging fails


@app.post("/audit", response_model=AIResponse)
async def audit_transaction(request: AIRequest):
    """
    Receives transaction data, analyzes it for policy violations,
    and returns a response with validity, reason, and risk score.
    """
    start_time = datetime.utcnow()

    try:
        if AUDIT_MODE == "llm" and LLM_API_KEY:
            # TODO: Implement real LLM integration
            # For now, fall back to local
            pass

        # Use local pattern-based analysis
        is_valid, reason, risk_score, checks_passed = analyze_local(request.transaction)

        # Add ML-based analysis
        ml_valid, ml_reason, ml_score = analyze_ml(request.transaction)

        combined_score = (risk_score + ml_score) / 2
        final_valid = is_valid and ml_valid
        final_reason = f"{reason}; ML: {ml_reason}"

        if ml_score > risk_score:
            combined_score = ml_score

        process_time = (datetime.utcnow() - start_time).total_seconds() * 1000
        response = AIResponse(
            valid=final_valid,
            reason=final_reason,
            risk_score=round(combined_score, 2),
            checks_passed=checks_passed if checks_passed else [],
        )

        # Update stats
        with audit_lock:
            stats["total_audited"] += 1
            if is_valid:
                stats["approved"] += 1
            else:
                stats["rejected"] += 1

        # Log the audit
        log_audit(request, response, process_time)

        return response

    except Exception as e:
        with audit_lock:
            stats["errors"] += 1
        # On error, allow transaction (fail-open for availability)
        return AIResponse(valid=True, reason=f"audit_error:{str(e)}")


@app.get("/health")
async def health_check():
    return {
        "status": "ok",
        "mode": AUDIT_MODE,
        "llm_configured": bool(LLM_API_KEY),
        "stats": stats,
    }


@app.get("/stats")
async def get_stats():
    """Get audit statistics"""
    with audit_lock:
        return stats


@app.get("/ai/stats")
async def get_ai_stats():
    """Get ML/AI statistics"""
    return get_global_stats()


@app.get("/ai/profile/{address}")
async def get_address_ai_profile(address: str):
    """Get AI-generated address profile"""
    return get_address_profile(address.lower())


@app.post("/ai/train")
async def train_model():
    """Retrain the ML model with current data"""
    return {"status": "training_not_required", "reason": "online_learning_enabled"}


@app.get("/logs")
async def get_logs(limit: int = 100):
    """Get recent audit logs"""
    with audit_lock:
        return audit_log[-limit:]


@app.post("/risklist/add")
async def add_high_risk_address(address: str):
    """Add address to high-risk list"""
    addr = address.lower().strip()
    if not re.match(r"^[a-f0-9]{64}$", addr):
        raise HTTPException(status_code=400, detail="Invalid address format")

    with audit_lock:
        HIGH_RISK_ADDRESSES.add(addr)

    # Persist to file
    try:
        os.makedirs(os.path.dirname(RISK_LIST_PATH), exist_ok=True)
        with open(RISK_LIST_PATH, "a") as f:
            f.write(addr + "\n")
    except Exception:
        pass

    return {"status": "added", "address": addr}


if __name__ == "__main__":
    import uvicorn

    port = int(os.getenv("PORT", "8081"))
    uvicorn.run(app, host="0.0.0.0", port=port)
