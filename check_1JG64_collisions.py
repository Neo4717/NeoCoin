import requests
import json
import time

def get_pk(addr):
    url = f"https://mempool.space/api/address/{addr}/txs"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            txs = resp.json()
            for tx in txs:
                for vin in tx.get('vin', []):
                    if vin.get('prevout', {}).get('scriptpubkey_address') == addr:
                        ss = vin.get('scriptsig', '')
                        if ss:
                            b = bytes.fromhex(ss)
                            if len(b) >= 33:
                                if b[-33] in [0x02, 0x03]: return b[-33:].hex()
                                if len(b) >= 65 and b[-65] == 0x04: return b[-65:].hex()
    except: pass
    return None

pk_1JG64 = "03b9ebca45d66cbc020feb99f3d3157d29ca302ab93dc81d2ff8551623a55c642e"

# Check dormant list
with open('bitcoin_crack/dormant_list.json', 'r') as f:
    dormant = json.load(f)

print(f"Checking 1JG64 PK against {len(dormant)} dormant addresses...")
for addr in dormant:
    # We can't fetch all PKs one by one (too slow)
    # But we can check if it's in the collision map!
    pass

# Load collision map
with open('data/pubkey_collision_map.json', 'r') as f:
    collisions = json.load(f)

if pk_1JG64 in collisions:
    print(f"!!! 1JG64 PK COLLISION DETECTED !!!")
    print(f"  Addresses: {collisions[pk_1JG64]}")
else:
    print("No collision found for 1JG64 in the map.")

