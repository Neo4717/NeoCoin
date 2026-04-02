import requests
import json
import binascii
import time

def extract_r_s(sig_hex):
    try:
        b = binascii.unhexlify(sig_hex)
        if b[0] == 0x30:
            r_len = b[3]
            r = b[4:4+r_len]
            if len(r) > 0 and r[0] == 0x00: r = r[1:]
            s_len = b[4+r_len+1]
            s = b[4+r_len+2:4+r_len+2+s_len]
            if len(s) > 0 and s[0] == 0x00: s = s[1:]
            return r.hex(), s.hex()
    except: pass
    return None, None

def get_all_txs(address):
    txs = []
    last_txid = None
    while True:
        url = f"https://mempool.space/api/address/{address}/txs"
        if last_txid: url += f"/chain/{last_txid}"
        try:
            resp = requests.get(url)
            if resp.status_code != 200: break
            new_txs = resp.json()
            if not new_txs: break
            txs.extend(new_txs)
            last_txid = new_txs[-1]['txid']
            if len(new_txs) < 25: break
            time.sleep(0.2)
        except: break
    return txs

shared_pk = "022038071e200488f3999bdd959fd2774038092cb513285a157b33f6a6c92cf3fc"
addresses = [
    "1P9fAFAsSLRmMu2P7wZ5CXDPRfLSWTy9N8",
    "18zuLTKQnLjp987LdxuYvjekYnNAvXif2b",
    "1HoDPH3wCSCiyGmSXX7xiadW2DayqaNaCo",
    "15HiQkbvQMoAzXyKdQbuCKTGDxTswYBUf5",
    "1AenFm1zSRkhtPHwZmP2UuRQbWpakD8cVZ",
    "13KYdPnzGh5H8exFY3FhUo9Rvvs6kKAcL8",
    "1EUJKGm3FB65rr5W9anAEoWA3m71WpDayZ",
    "18cKGtwdQHmnDXD6w6AhBhHsmxnK8gsVHf",
    "19DdkMxutkLGY67REFPLu51imfxG9CUJLD"
]

print(f"Auditing family for PK {shared_pk}...")
r_map = {}

for addr in addresses:
    print(f"  Processing {addr}...")
    txs = get_all_txs(addr)
    print(f"    Found {len(txs)} transactions.")
    for tx in txs:
        for vin in tx.get('vin', []):
            prevout = vin.get('prevout')
            if prevout and prevout.get('scriptpubkey_address') == addr:
                sig_hex = ""
                ss_asm = vin.get('scriptsig_asm', '')
                for part in ss_asm.split(' '):
                    if len(part) > 100:
                        sig_hex = part
                        break
                if not sig_hex:
                    witness = vin.get('witness', [])
                    if witness: sig_hex = witness[0]
                
                if sig_hex:
                    r, s = extract_r_s(sig_hex)
                    if r:
                        if r in r_map:
                            prev = r_map[r]
                            if prev['txid'] != tx['txid']:
                                print(f"!!! CRITICAL: NONCE REUSE DETECTED !!!")
                                print(f"  R: {r}")
                                print(f"  TX1: {prev['txid']} (Addr: {prev['addr']})")
                                print(f"  TX2: {tx['txid']} (Addr: {addr})")
                        r_map[r] = {'txid': tx['txid'], 'addr': addr, 's': s}

print("Deep Family Audit Finished.")
