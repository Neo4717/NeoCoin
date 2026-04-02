import requests
import json
import time

def fetch_txs(address):
    print(f"Fetching transactions for {address}...")
    url = f"https://mempool.space/api/address/{address}/txs"
    response = requests.get(url)
    if response.status_code == 200:
        return response.json()
    else:
        print(f"Error {response.status_code}")
        return None

address = "bc1qks8zrshwmu3m8vgqdzwl2u8jjfgnvgjlezwqcd"
txs = fetch_txs(address)
if txs:
    with open(f"data/raw_txs_{address[:6]}.json", "w") as f:
        json.dump(txs, f)
    print(f"Saved {len(txs)} transactions.")
