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

def get_txs(addr):
    url = f"https://mempool.space/api/address/{addr}/txs"
    try:
        resp = requests.get(url)
        return resp.json() if resp.status_code == 200 else []
    except: return []

# Load all targets
targets = set()
for f in ['data/puzzle_addresses.json', 'bitcoin_crack/dormant_list.json']:
    try:
        with open(f, 'r') as f_in:
            for a in json.load(f_in): targets.add(a)
    except: pass

funders = ["145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ", "1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu", "bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h", "bc1qks8zrshwmu3m8vgqdzwl2u8jjfgnvgjlezwqcd"]
for f in funders: targets.add(f)

print(f"Total targets: {len(targets)}")

r_map = {} # r -> (txid, addr, s)

for i, addr in enumerate(targets):
    if i % 50 == 0: print(f"Progress: {i}/{len(targets)}")
    txs = get_txs(addr)
    for tx in txs:
        for vin in tx.get('vin', []):
            prevout = vin.get('prevout');
            if prevout and prevout.get('scriptpubkey_address') == addr:
                sig_hex = ""
                # Try scriptsig
                ss_asm = vin.get('scriptsig_asm', '')
                for part in ss_asm.split(' '):
                    if len(part) > 100: # Heuristic for sig
                        sig_hex = part
                        break
                if not sig_hex:
                    # Try witness
                    witness = vin.get('witness', [])
                    if witness: sig_hex = witness[0]
                
                if sig_hex:
                    r, s = extract_r_s(sig_hex)
                    if r:
                        if r in r_map:
                            prev_txid, prev_addr, prev_s = r_map[r]
                            if prev_txid != tx['txid']:
                                print(f"!!! R-COLLISION !!!")
                                print(f"  R: {r}")
                                print(f"  TX1: {prev_txid} (Addr: {prev_addr}, S: {prev_s})")
                                print(f"  TX2: {tx['txid']} (Addr: {addr}, S: {s})")
                        r_map[r] = (tx['txid'], addr, s)
    time.sleep(0.1)

print("Mega Audit Finished.")
