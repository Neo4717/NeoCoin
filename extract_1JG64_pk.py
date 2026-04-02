import json
import binascii
def extract_pk(scriptsig):
    try:
        b = binascii.unhexlify(scriptsig)
        for i in range(len(b)):
            if b[i] == 0x04 and i+64 < len(b): return binascii.hexlify(b[i : i+65]).decode()
            if (b[i] == 0x02 or b[i] == 0x03) and i+32 < len(b): return binascii.hexlify(b[i : i+33]).decode()
    except: pass
    return None
with open('data/raw_txs_1JG64.json', 'r') as f:
    txs = json.load(f)
for tx in txs:
    for vin in tx.get('vin', []):
        if vin.get('prevout', {}).get('scriptpubkey_address') == '1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu':
            pk = extract_pk(vin.get('scriptsig', ''))
            if pk:
                print(f"FOUND PK for 1JG64: {pk}")
                break
