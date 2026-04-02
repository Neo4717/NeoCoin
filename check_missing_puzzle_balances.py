import requests
import json
import time

with open('data/puzzle_addresses.json', 'r') as f:
    addresses = json.load(f)

for i in range(60, 80):
    addr = addresses[i]
    print(f"[#{i+1}] Checking {addr}...")
    url = f"https://mempool.space/api/address/{addr}"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            data = resp.json()
            balance = data['chain_stats']['funded_txo_sum'] - data['chain_stats']['spent_txo_sum']
            print(f"  BALANCE: {balance} sats")
        time.sleep(0.5)
    except Exception as e:
        print(f"  Error: {e}")
