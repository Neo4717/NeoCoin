import requests
import json
import binascii
import time

def extract_r(scriptsig):
    try:
        b = binascii.unhexlify(scriptsig)
        for i in range(len(b)-5):
            if b[i] == 0x30 and b[i+2] == 0x02:
                rLen = b[i+3]
                if i+4+rLen <= len(b):
                    r = b[i+4 : i+4+rLen]
                    if len(r) > 0 and r[0] == 0x00: r = r[1:]
                    return r.hex()
    except: pass
    return None

def extract_r_from_witness(witness):
    try:
        # witness[0] is usually the signature
        b = binascii.unhexlify(witness[0])
        return extract_r(witness[0])
    except: return None

with open('data/pubkey_collision_map.json', 'r') as f:
    collision_data = json.load(f)

# Collect all addresses that are part of a collision
addresses_to_check = set()
for pk, addrs in collision_data.items():
    if len(addrs) > 1:
        for addr in addrs:
            addresses_to_check.add(addr)

print(f"Total unique addresses involved in collisions: {len(addresses_to_check)}")

r_values = {} # r -> list of (txid, address, z)

for addr in addresses_to_check:
    print(f"Fetching transactions for {addr}...")
    url = f"https://mempool.space/api/address/{addr}/txs"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            txs = resp.json()
            for tx in txs:
                for vin in tx.get('vin', []):
                    if vin.get('prevout', {}).get('scriptpubkey_address') == addr:
                        r = extract_r(vin.get('scriptsig', ''))
                        if not r:
                            witness = vin.get('witness', [])
                            if witness:
                                r = extract_r(witness[0])
                        
                        if r:
                            if r in r_values:
                                # Found a collision!
                                for prev in r_values[r]:
                                    if prev['txid'] != tx['txid']:
                                        print(f"!!! NONCE REUSE DETECTED !!!")
                                        print(f"  R: {r}")
                                        print(f"  TX1: {prev['txid']} (Address: {prev['addr']})")
                                        print(f"  TX2: {tx['txid']} (Address: {addr})")
                                        # To fully recover, we'd need 'z' (the message hash)
                                        # But finding the R collision is the hardest part.
                                r_values[r].append({'txid': tx['txid'], 'addr': addr})
                            else:
                                r_values[r] = [{'txid': tx['txid'], 'addr': addr}]
        time.sleep(0.5)
    except Exception as e:
        print(f"  Error fetching {addr}: {e}")

print("Audit complete.")
