import json
import binascii

def extract_r(scriptsig):
    try:
        b = binascii.unhexlify(scriptsig)
        idx = b.find(b'\x30')
        if idx != -1:
            sig = b[idx+2:]
            if len(sig) > 2 and sig[0] == 0x02:
                len_r = sig[1]
                if len(sig) >= 2 + len_r:
                    return binascii.hexlify(sig[2:2+len_r]).decode()
    except:
        pass
    return None

with open('data/raw_txs_1GSMG1.json', 'r') as f:
    txs = json.load(f)

r_map = {}
for tx in txs:
    for vin in tx.get('vin', []):
        ss = vin.get('scriptsig', '')
        if ss:
            r = extract_r(ss)
            if r:
                if r in r_map:
                    print(f"!!! NONCE REUSE FOUND !!! R: {r}")
                    print(f"  TX1: {r_map[r]}")
                    print(f"  TX2: {tx['txid']}")
                r_map[r] = tx['txid']
