import json
import binascii

def extract_pk(scriptsig_asm, witness):
    # Try witness (P2WPKH)
    if witness and len(witness) == 2:
        pk = witness[1]
        if len(pk) in [66, 130]: return pk
    
    # Try scriptsig_asm (Legacy)
    parts = scriptsig_asm.split(' ')
    for part in parts:
        if len(part) in [66, 130]:
            if part.startswith(('02', '03', '04')): return part
    return None

with open('data/raw_txs_1GSMG1.json', 'r') as f:
    txs = json.load(f)

pk_to_addrs = {}

for tx in txs:
    for vin in tx.get('vin', []):
        addr = vin.get('prevout', {}).get('scriptpubkey_address')
        if addr:
            pk = extract_pk(vin.get('scriptsig_asm', ''), vin.get('witness', []))
            if pk:
                if pk not in pk_to_addrs: pk_to_addrs[pk] = set()
                pk_to_addrs[pk].add(addr)

for pk, addrs in pk_to_addrs.items():
    if len(addrs) > 1:
        print(f"!!! PK COLLISION DETECTED !!!")
        print(f"  PK: {pk}")
        print(f"  Addresses: {list(addrs)}")

