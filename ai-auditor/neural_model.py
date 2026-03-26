"""
NeoCoin Neural Network Model
Fraud detection using scikit-learn MLPClassifier
"""

import numpy as np
import joblib
import os
from typing import Tuple, List, Dict
import hashlib
import threading


class NeuralFraudDetector:
    def __init__(self):
        self.model = None
        self.feature_names = [
            "amount_zscore",
            "interval_zscore",
            "sender_freq",
            "recipient_freq",
            "data_entropy",
            "tx_count",
            "avg_amount",
            "volatility",
        ]
        self._init_model()

    def _init_model(self):
        model_path = "/app/data/fraud_model.joblib"
        if os.path.exists(model_path):
            try:
                self.model = joblib.load(model_path)
                return
            except:
                pass

        from sklearn.neural_network import MLPClassifier
        from sklearn.preprocessing import StandardScaler

        self.model = MLPClassifier(
            hidden_layer_sizes=(64, 32, 16),
            activation="relu",
            solver="adam",
            max_iter=500,
            random_state=42,
            early_stopping=True,
        )
        self.scaler = StandardScaler()

        X = np.random.randn(1000, 8)
        y = (X[:, 0] + X[:, 1] > 1).astype(int)

        X_scaled = self.scaler.fit_transform(X)
        self.model.fit(X_scaled, y)

        try:
            os.makedirs("/app/data", exist_ok=True)
            joblib.dump(self.model, model_path)
        except:
            pass

    def extract_features(
        self,
        sender: str,
        recipient: str,
        amount: float,
        data: str,
        interval_seconds: float,
        tx_count: int,
        avg_amount: float,
    ) -> np.ndarray:
        from advanced_ai import calculate_risk_entropy, get_std, get_mean
        from advanced_ai import stats

        amount_zscore = 0
        interval_zscore = 0

        with stats.lock:
            if len(stats.amounts) > 1:
                mean_amt = get_mean(stats.amounts)
                std_amt = get_std(stats.amounts)
                if std_amt > 0:
                    amount_zscore = (amount - mean_amt) / std_amt

            if len(stats.intervals) > 1:
                mean_int = get_mean(stats.intervals)
                std_int = get_std(stats.intervals)
                if std_int > 0:
                    interval_zscore = (interval_seconds - mean_int) / std_int

        sender_count = stats.address_tx_counts.get(sender.lower(), 0)
        recipient_count = stats.address_tx_counts.get(recipient.lower(), 0)
        data_entropy = calculate_risk_entropy(data)

        volatility = 0.0
        with stats.lock:
            if sender in stats.address_volumes:
                vol = stats.address_volumes[sender]
                cnt = stats.address_tx_counts.get(sender, 1)
                if cnt > 1:
                    volatility = vol / cnt

        features = np.array(
            [
                amount_zscore,
                interval_zscore,
                sender_count / 100.0,
                recipient_count / 100.0,
                data_entropy / 8.0,
                tx_count / 100.0,
                avg_amount / 1000000.0,
                volatility / 1000000.0,
            ]
        )

        return features

    def predict(self, features: np.ndarray) -> Tuple[bool, float]:
        if self.model is None:
            return True, 0.0

        try:
            X = features.reshape(1, -1)
            prob = self.model.predict_proba(X)[0][1]
            is_fraud = prob > 0.7
            return not is_fraud, prob * 100.0
        except Exception as e:
            return True, 0.0

    def train(self, features: np.ndarray, labels: np.ndarray):
        if self.model is None:
            return

        try:
            X_scaled = self.scaler.fit_transform(features)
            self.model.fit(X_scaled, labels)

            model_path = "/app/data/fraud_model.joblib"
            os.makedirs("/app/data", exist_ok=True)
            joblib.dump(self.model, model_path)
        except:
            pass

    def get_risk_level(self, score: float) -> str:
        if score >= 70:
            return "critical"
        elif score >= 50:
            return "high"
        elif score >= 30:
            return "medium"
        elif score >= 10:
            return "low"
        return "minimal"


detector = NeuralFraudDetector()


def analyze_with_ml(
    sender: str,
    recipient: str,
    amount: float,
    data: str,
    interval_seconds: float,
    tx_count: int,
    avg_amount: float,
) -> dict:
    features = detector.extract_features(
        sender, recipient, amount, data, interval_seconds, tx_count, avg_amount
    )
    is_safe, score = detector.predict(features)

    return {
        "is_safe": is_safe,
        "fraud_probability": round(score, 2),
        "risk_level": detector.get_risk_level(score),
        "model_version": "1.0.0",
    }


def retrain_model(feedback_data: List[dict]):
    X = []
    y = []

    for item in feedback_data:
        features = np.array(item["features"])
        label = 1 if item["is_fraud"] else 0
        X.append(features)
        y.append(label)

    if len(X) > 10:
        detector.train(np.array(X), np.array(y))


_fraud_trends = {
    "defi_exploits": 0,
    "phishing": 0,
    "rugpulls": 0,
    "mixer_usage": 0,
    "suspicious_patterns": 0,
}

_trends_lock = threading.Lock()


def get_fraud_trends() -> Dict:
    with _trends_lock:
        total = sum(_fraud_trends.values())
        if total == 0:
            return {
                "defi_exploits": 0,
                "phishing": 0,
                "rugpulls": 0,
                "mixer_usage": 0,
                "suspicious_patterns": 0,
                "total_alerts": 0,
                "risk_level": "low",
            }

        return {
            "defi_exploits": _fraud_trends["defi_exploits"],
            "phishing": _fraud_trends["phishing"],
            "rugpulls": _fraud_trends["rugpulls"],
            "mixer_usage": _fraud_trends["mixer_usage"],
            "suspicious_patterns": _fraud_trends["suspicious_patterns"],
            "total_alerts": total,
            "risk_level": "high" if total > 100 else "medium" if total > 50 else "low",
        }


def record_fraud_alert(alert_type: str):
    with _trends_lock:
        if alert_type in _fraud_trends:
            _fraud_trends[alert_type] += 1
        else:
            _fraud_trends["suspicious_patterns"] += 1
