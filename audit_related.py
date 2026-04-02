import requests
import json
import binascii
import hashlib

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

addresses = [
    "145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ",
    "1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu",
    "bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h",
    "bc1qks8zrshwmu3m8vgqdzwl2u8jjfgnvgjlezwqcd",
    "1GNB9PdRPtc4R7cYyLTuUmkbBrfiXoW7Kp",
    "1PM8huQVFSirUT7eAwNm3rBBYTsDRzCaf3"
]

r_values = {}

for addr in addresses:
    print(f"Auditing {addr}...")
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
                            # Try witness for Segwit
                            witness = vin.get('witness', [])
                            if witness:
                                r = extract_r(witness[0])
                        if r:
                            if r in r_values:
                                if r_values[r]['txid'] != tx['txid']:
                                    print(f"!!! NONCE REUSE FOUND !!! R: {r}")
                                    print(f"  TX1: {r_values[r]['txid']} (Addr: {r_values[r]['addr']})")
                                    print(f"  TX2: {tx['txid']} (Addr: {addr})")
                            r_values[r] = {'txid': tx['txid'], 'addr': addr}
    except Exception as e:
        print(f"  Error: {e}")

print("Audit complete.")
