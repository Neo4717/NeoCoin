import requests
import json
import time

with open('data/puzzle_addresses.json', 'r') as f:
    addresses = json.load(f)

total_balance = 0
for i, addr in enumerate(addresses):
    print(f"[{i+1}/{len(addresses)}] Checking {addr}...")
    url = f"https://mempool.space/api/address/{addr}"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            data = resp.json()
            balance = data['chain_stats']['funded_txo_sum'] - data['chain_stats']['spent_txo_sum']
            if balance > 0:
                print(f"  BALANCE in {addr}: {balance} sats")
                total_balance += balance
        time.sleep(0.5)
    except Exception as e:
        # print(f"  Error: {e}")
        pass

print(f"\nTotal Balance: {total_balance} sats")
