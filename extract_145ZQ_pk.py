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
with open('data/raw_txs_145ZQ.json', 'r') as f:
    txs = json.load(f)
for tx in txs:
    for vin in tx.get('vin', []):
        if vin.get('prevout', {}).get('scriptpubkey_address') == '145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ':
            pk = extract_real_pk(vin.get('scriptsig', ''))
            if pk:
                print(f"FOUND PK for 145ZQ: {pk}")
                break
