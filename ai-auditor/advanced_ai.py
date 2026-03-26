"""
NeoCoin AI Auditor - Advanced ML Features
Statistical anomaly detection and behavioral analysis
"""

import threading
import math
from collections import defaultdict
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple
import hashlib
import json

lock = threading.Lock()


class TransactionStats:
    def __init__(self):
        self.amounts: List[float] = []
        self.intervals: List[float] = []
        self.address_tx_counts: Dict[str, int] = defaultdict(int)
        self.address_volumes: Dict[str, float] = defaultdict(float)
        self.last_tx_time: Dict[str, datetime] = {}

    def add_transaction(self, sender: str, amount: float, timestamp: datetime):
        with lock:
            self.amounts.append(amount)
            self.address_tx_counts[sender] += 1
            self.address_volumes[sender] += amount

            if sender in self.last_tx_time:
                interval = (timestamp - self.last_tx_time[sender]).total_seconds()
                if interval > 0:
                    self.intervals.append(interval)
            self.last_tx_time[sender] = timestamp

            if len(self.amounts) > 10000:
                self.amounts = self.amounts[-5000:]
            if len(self.intervals) > 10000:
                self.intervals = self.intervals[-5000:]


stats = TransactionStats()


def get_mean(values: List[float]) -> float:
    return sum(values) / len(values) if values else 0.0


def get_std(values: List[float]) -> float:
    if len(values) < 2:
        return 0.0
    mean = get_mean(values)
    variance = sum((x - mean) ** 2 for x in values) / len(values)
    return math.sqrt(variance)


def calculate_z_score(value: float, values: List[float]) -> float:
    mean = get_mean(values)
    std = get_std(values)
    if std == 0:
        return 0.0
    return (value - mean) / std


def detect_anomaly_zscore(amount: float, sender: str) -> Tuple[bool, str]:
    with lock:
        if len(stats.amounts) < 10:
            return False, "insufficient_data"

        z_score = calculate_z_score(amount, stats.amounts)

        if abs(z_score) > 5:
            return True, f"extreme_amount_zscore:{z_score:.2f}"
        elif abs(z_score) > 3:
            return True, f"unusual_amount_zscore:{z_score:.2f}"

        sender_count = stats.address_tx_counts.get(sender, 0)
        if sender_count > 100:
            z_score_count = calculate_z_score(
                sender_count, list(stats.address_tx_counts.values())
            )
            if z_score_count > 4:
                return True, f"high_frequency_sender:{sender_count}"

    return False, ""


def analyze_frequency(sender: str, timestamp: datetime) -> Tuple[bool, str]:
    with lock:
        if sender not in stats.last_tx_time:
            return False, ""

        interval = (timestamp - stats.last_tx_time[sender]).total_seconds()

        if len(stats.intervals) >= 10:
            mean_interval = get_mean(stats.intervals)
            if interval < mean_interval * 0.1 and mean_interval > 60:
                return True, f"rapid_succession:{interval:.1f}s"

    return False, ""


def detect_wash_trading(sender: str, recipient: str, amount: float) -> bool:
    if sender == recipient:
        return True

    with lock:
        sender_volume = stats.address_volumes.get(sender, 0)
        if sender_volume > amount * 10 and stats.address_tx_counts.get(sender, 0) > 5:
            return True

    return False


def calculate_risk_entropy(data: str) -> float:
    if not data:
        return 0.0

    char_freq = defaultdict(int)
    for c in data:
        char_freq[c] += 1

    entropy = 0.0
    for count in char_freq.values():
        p = count / len(data)
        if p > 0:
            entropy -= p * math.log2(p)

    return entropy


def detect_data_anomaly(data: str) -> Tuple[bool, str]:
    if not data:
        return False, ""

    if len(data) > 10000:
        return True, "excessive_data_length"

    entropy = calculate_risk_entropy(data)
    if entropy > 7.5:
        return True, f"high_entropy:{entropy:.2f}"

    if len(set(data)) < len(data) * 0.3:
        return True, "low_diversity_data"

    return False, ""


class BehaviorClassifier:
    def __init__(self):
        self.normal_patterns = {
            "small_regular": 0.0,
            "large_rare": 0.0,
            "burst": 0.0,
            "dormant": 0.0,
        }

    def classify(self, sender: str, amount: float, timestamp: datetime) -> str:
        with lock:
            tx_count = stats.address_tx_counts.get(sender, 0)
            avg_amount = stats.address_volumes.get(sender, 0) / max(tx_count, 1)

            if tx_count == 0:
                return "new_address"

            if amount > avg_amount * 5:
                return "large_unusual"
            elif amount < avg_amount * 0.2 and tx_count > 10:
                return "dusting"
            elif tx_count > 50:
                return "high_volume"
            else:
                return "normal"


classifier = BehaviorClassifier()


def ml_score_transaction(
    sender: str, recipient: str, amount: float, data: str, timestamp: datetime
) -> float:
    score = 0.0

    is_anomaly, _ = detect_anomaly_zscore(amount, sender)
    if is_anomaly:
        score += 25.0

    is_freq, _ = analyze_frequency(sender, timestamp)
    if is_freq:
        score += 20.0

    if detect_wash_trading(sender, recipient, amount):
        score += 30.0

    is_data, _ = detect_data_anomaly(data)
    if is_data:
        score += 15.0

    behavior = classifier.classify(sender, amount, timestamp)
    if behavior in ["large_unusual", "dusting"]:
        score += 15.0
    elif behavior == "new_address":
        score += 5.0

    return min(score, 100.0)


def record_transaction(sender: str, recipient: str, amount: float):
    stats.add_transaction(sender, amount, datetime.utcnow())


def get_address_profile(address: str) -> Dict:
    with lock:
        return {
            "tx_count": stats.address_tx_counts.get(address, 0),
            "total_volume": stats.address_volumes.get(address, 0),
            "last_seen": stats.last_tx_time.get(address, None).isoformat()
            if stats.last_tx_time.get(address)
            else None,
            "behavior": classifier.classify(address, 0, datetime.utcnow()),
        }


def get_global_stats() -> Dict:
    with lock:
        return {
            "total_txs": len(stats.amounts),
            "unique_addresses": len(stats.address_tx_counts),
            "avg_amount": get_mean(stats.amounts),
            "avg_interval": get_mean(stats.intervals),
            "amount_std": get_std(stats.amounts),
        }


def analyze_network_topology(sender: str, recipient: str) -> Dict:
    with lock:
        sender_connections = stats.address_tx_counts.get(sender, 0)
        recipient_connections = stats.address_tx_counts.get(recipient, 0)

        sender_volume = stats.address_volumes.get(sender, 0)
        recipient_volume = stats.address_volumes.get(recipient, 0)

        risk_score = 0.0
        is_known_mixer = False

        if sender_connections > 100:
            risk_score += 20
        if recipient_connections > 100:
            risk_score += 20

        if sender_volume > 10000000 or recipient_volume > 10000000:
            risk_score += 15

        return {
            "risk_score": risk_score,
            "sender_connections": sender_connections,
            "recipient_connections": recipient_connections,
            "is_known_mixer": is_known_mixer,
        }


def get_address_risk_history(address: str) -> Dict:
    with lock:
        tx_count = stats.address_tx_counts.get(address, 0)
        volume = stats.address_volumes.get(address, 0)

        if tx_count == 0:
            return None

        avg_tx_size = volume / tx_count if tx_count > 0 else 0

        risk_score = 0.0
        if tx_count < 3:
            risk_score += 20
        if avg_tx_size > 1000000:
            risk_score += 15

        return {
            "tx_count": tx_count,
            "volume": volume,
            "avg_tx_size": avg_tx_size,
            "risk_score": risk_score,
        }


def calculate_entropy_score(data: str) -> float:
    if not data:
        return 0.0

    char_freq = defaultdict(int)
    for c in data:
        char_freq[c] += 1

    entropy = 0.0
    for count in char_freq.values():
        p = count / len(data)
        if p > 0:
            entropy -= p * math.log2(p)

    return entropy


def get_address_connections(address: str) -> Dict:
    with lock:
        connections = []

        for addr in stats.address_tx_counts:
            if addr != address and (
                addr.startswith(address[:8]) or address.startswith(addr[:8])
            ):
                connections.append(
                    {
                        "address": addr,
                        "tx_count": stats.address_tx_counts[addr],
                        "volume": stats.address_volumes.get(addr, 0),
                    }
                )

        return {
            "address": address,
            "connections": connections[:10],
            "total_connections": len(connections),
        }
