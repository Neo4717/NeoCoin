import requests
import json
import binascii
import time

def decode_hex(hex_str):
    try:
        return binascii.unhexlify(hex_str).decode('ascii', errors='replace')
    except:
        return None

with open('data/puzzle_addresses.json', 'r') as f:
    addresses = json.load(f)

for i, addr in enumerate(addresses):
    print(f"[{i+1}/{len(addresses)}] Scanning {addr}...")
    url = f"https://mempool.space/api/address/{addr}/txs"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            txs = resp.json()
            for tx in txs:
                for vout in tx.get('vout', []):
                    asm = vout.get('scriptpubkey_asm', '')
                    if 'OP_RETURN' in asm:
                        hex_data = asm.split(' ')[-1]
                        decoded = decode_hex(hex_data)
                        if decoded:
                            print(f"  FOUND CLUE in {addr}: {decoded}")
        time.sleep(0.5)
    except Exception as e:
        print(f"  Error: {e}")
