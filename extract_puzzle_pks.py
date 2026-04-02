import requests
import json
import binascii
import time

def extract_pk(scriptsig):
    b = binascii.unhexlify(scriptsig)
    for i in range(len(b)):
        if b[i] == 0x04 and i+64 < len(b): return binascii.hexlify(b[i : i+65]).decode()
        if (b[i] == 0x02 or b[i] == 0x03) and i+32 < len(b): return binascii.hexlify(b[i : i+33]).decode()
    return None

with open('data/puzzle_addresses.json', 'r') as f:
    addresses = json.load(f)

pk_map = {}
for i, addr in enumerate(addresses):
    print(f"[{i+1}/{len(addresses)}] Scanning {addr}...")
    url = f"https://mempool.space/api/address/{addr}/txs"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            txs = resp.json()
            for tx in txs:
                for vin in tx.get('vin', []):
                    # We want to find the public key of the address itself
                    # This only works if it was used as an input
                    if vin.get('prevout', {}).get('scriptpubkey_address') == addr:
                        pk = extract_pk(vin.get('scriptsig', ''))
                        if pk:
                            pk_map[addr] = pk
                            print(f"  FOUND PK for {addr}: {pk}")
                            break
                if addr in pk_map: break
        time.sleep(0.5)
    except Exception as e:
        pass

with open('data/puzzle_pks.json', 'w') as f:
    json.dump(pk_map, f)
