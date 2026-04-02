import requests
import json
import binascii
import time

def extract_real_pk(vin, addr):
    try:
        if vin.get('prevout', {}).get('scriptpubkey_address') == addr:
            # Check scriptsig
            ss = vin.get('scriptsig', '')
            if ss:
                b = binascii.unhexlify(ss)
                if len(b) >= 33:
                    if b[-33] in [0x02, 0x03]: return b[-33:].hex()
                    if len(b) >= 65 and b[-65] == 0x04: return b[-65:].hex()
            # Check witness
            witness = vin.get('witness', [])
            if len(witness) == 2:
                pk = witness[1]
                if len(pk) in [66, 130]: return pk
    except: pass
    return None

def get_pk(addr):
    url = f"https://mempool.space/api/address/{addr}/txs"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            txs = resp.json()
            for tx in txs:
                for vin in tx.get('vin', []):
                    pk = extract_real_pk(vin, addr)
                    if pk: return pk
    except: pass
    return None

puzzle_files = ['data/puzzle_addresses.json']
funders = ["145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ", "1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu", "bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h", "bc1qks8zrshwmu3m8vgqdzwl2u8jjfgnvgjlezwqcd"]

with open('data/puzzle_addresses.json', 'r') as f:
    puzzle_addrs = json.load(f)

funder_pks = {}
for f in funders:
    pk = get_pk(f)
    if pk:
        funder_pks[f] = pk
        print(f"Funder {f} PK: {pk}")

for addr in puzzle_addrs:
    pk = get_pk(addr)
    if pk:
        for f_addr, f_pk in funder_pks.items():
            if pk == f_pk:
                print(f"!!! KEY COLLISION FOUND !!!")
                print(f"  Puzzle Addr: {addr}")
                print(f"  Funder Addr: {f_addr}")
                print(f"  Shared PK: {pk}")
    time.sleep(0.2)

