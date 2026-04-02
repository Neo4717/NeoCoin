import json
from collections import Counter

with open('data/full_audit_1GSMG1.json', 'r') as f:
    data = json.load(f)

scriptsigs = []
tx_map = {} # scriptsig -> list of txids

for item in data:
    for vin in item.get('vin', []):
        ss = vin.get('scriptsig')
        if ss:
            scriptsigs.append(ss)
            if ss not in tx_map: tx_map[ss] = []
            tx_map[ss].append(item['txid'])

counts = Counter(scriptsigs)
for ss, count in counts.items():
    if count > 1:
        print(f"!!! DUPLICATE SCRIPTSIG FOUND ({count} times) !!!")
        print(f"  ScriptSig: {ss[:50]}...")
        print(f"  TXIDs: {tx_map[ss]}")

