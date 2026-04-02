import requests
import json
import binascii
import time

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

all_addresses = set()

# Load from various sources
files = ['data/puzzle_addresses.json', 'bitcoin_crack/dormant_list.json']
for f_path in files:
    try:
        with open(f_path, 'r') as f:
            addrs = json.load(f)
            for a in addrs: all_addresses.add(a)
    except: pass

# Also include the ones we found earlier
all_addresses.add("145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ")
all_addresses.add("1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu")
all_addresses.add("bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h")
all_addresses.add("bc1qks8zrshwmu3m8vgqdzwl2u8jjfgnvgjlezwqcd")

print(f"Total addresses to audit: {len(all_addresses)}")

r_map = {} # r -> {'txid', 'addr'}
pubkey_map = {} # pubkey -> list of addresses

for i, addr in enumerate(all_addresses):
    if i % 20 == 0: print(f"Progress: {i}/{len(all_addresses)}")
    url = f"https://mempool.space/api/address/{addr}/txs"
    try:
        resp = requests.get(url)
        if resp.status_code == 200:
            txs = resp.json()
            for tx in txs:
                for vin in tx.get('vin', []):
                    if vin.get('prevout', {}).get('scriptpubkey_address') == addr:
                        ss = vin.get('scriptsig', '')
                        r = extract_r(ss)
                        if not r:
                            witness = vin.get('witness', [])
                            if witness: r = extract_r(witness[0])
                        
                        if r:
                            if r in r_map and r_map[r]['txid'] != tx['txid']:
                                print(f"!!! R-COLLISION !!! R: {r}")
                                print(f"  TX1: {r_map[r]['txid']} (Addr: {r_map[r]['addr']})")
                                print(f"  TX2: {tx['txid']} (Addr: {addr})")
                            r_map[r] = {'txid': tx['txid'], 'addr': addr}
        time.sleep(0.3)
    except: pass

