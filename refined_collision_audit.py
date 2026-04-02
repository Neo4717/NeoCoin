import requests
import json
import binascii
import time

def extract_r(sig_hex):
    try:
        b = binascii.unhexlify(sig_hex)
        # Check for DER encoding
        if b[0] == 0x30:
            r_len = b[3]
            r = b[4:4+r_len]
            if len(r) > 0 and r[0] == 0x00: r = r[1:]
            return r.hex()
    except: pass
    return None

def get_all_txs(address):
    txs = []
    last_txid = None
    while True:
        url = f"https://mempool.space/api/address/{address}/txs"
        if last_txid:
            url += f"/chain/{last_txid}"
        try:
            resp = requests.get(url)
            if resp.status_code != 200: break
            new_txs = resp.json()
            if not new_txs: break
            txs.extend(new_txs)
            last_txid = new_txs[-1]['txid']
            if len(new_txs) < 25: break
            time.sleep(0.5)
        except: break
    return txs

with open('data/pubkey_collision_map.json', 'r') as f:
    collisions = json.load(f)

for pk, addresses in collisions.items():
    if len(addresses) < 2: continue
    
    print(f"Checking Collision for PK {pk} ({len(addresses)} addresses)...")
    r_map = {}
    
    for addr in addresses:
        print(f"  Fetching all TXs for {addr}...")
        txs = get_all_txs(addr)
        for tx in txs:
            for vin in tx.get('vin', []):
                if vin.get('prevout', {}).get('scriptpubkey_address') == addr:
                    # Try scriptsig
                    ss = vin.get('scriptsig', '')
                    if ss:
                        # ScriptSig usually contains <sig> <pubkey>
                        # We need to extract the sig part
                        parts = vin.get('scriptsig_asm', '').split(' ')
                        for part in parts:
                            r = extract_r(part)
                            if r:
                                if r in r_map and r_map[r] != tx['txid']:
                                    print(f"!!! NONCE REUSE !!! R: {r}")
                                    print(f"  TX1: {r_map[r]}")
                                    print(f"  TX2: {tx['txid']}")
                                r_map[r] = tx['txid']
                    # Try witness
                    witness = vin.get('witness', [])
                    if witness:
                        r = extract_r(witness[0])
                        if r:
                            if r in r_map and r_map[r] != tx['txid']:
                                print(f"!!! NONCE REUSE (Witness) !!! R: {r}")
                                print(f"  TX1: {r_map[r]}")
                                print(f"  TX2: {tx['txid']}")
                            r_map[r] = tx['txid']
