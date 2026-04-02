import requests
import json
import binascii

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
    url = f"https://mempool.space/api/address/{address}/txs"
    return requests.get(url).json()

addr = "12CRBZwUsrS1nqTdx2suKKiDobPWJf7XFt"
txs = get_all_txs(addr)
r_map = {}

for tx in txs:
    for vin in tx.get('vin', []):
        if vin.get('prevout', {}).get('scriptpubkey_address') == addr:
            ss_asm = vin.get('scriptsig_asm', '')
            for part in ss_asm.split(' '):
                if len(part) > 100:
                    r, s = extract_r_s(part)
                    if r:
                        if r in r_map and r_map[r] != tx['txid']:
                            print(f"!!! NONCE REUSE !!! R: {r}")
                            print(f"  TX1: {r_map[r]}")
                            print(f"  TX2: {tx['txid']}")
                        r_map[r] = tx['txid']
