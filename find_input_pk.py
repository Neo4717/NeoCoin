import requests
import json
import binascii

def extract_real_pk(scriptsig):
    try:
        b = binascii.unhexlify(scriptsig)
        if len(b) >= 33:
            if b[-33] in [0x02, 0x03]: return binascii.hexlify(b[-33:]).decode()
            if len(b) >= 65 and b[-65] == 0x04: return binascii.hexlify(b[-65:]).decode()
    except: pass
    return None

address = "1PWo3JeB9jrGwfHDNpdGK54CRas7fsVzXU"
url = f"https://mempool.space/api/address/{address}/txs"
txs = requests.get(url).json()

for tx in txs:
    for vin in tx.get('vin', []):
        if vin.get('prevout', {}).get('scriptpubkey_address') == address:
            pk = extract_real_pk(vin.get('scriptsig', ''))
            if pk:
                print(f"TX {tx['txid']} spent from {address} using PK {pk}")
