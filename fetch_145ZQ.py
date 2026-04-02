import requests
import json

def fetch_txs(address):
    url = f"https://mempool.space/api/address/{address}/txs"
    return requests.get(url).json()

txs = fetch_txs("145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ")
with open("data/raw_txs_145ZQ.json", "w") as f:
    json.dump(txs, f)
