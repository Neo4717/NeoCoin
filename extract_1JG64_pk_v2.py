import json
import binascii
def extract_real_pk(scriptsig):
    try:
        b = binascii.unhexlify(scriptsig)
        # PK is usually at the end
        if len(b) >= 33:
            # Check for compressed PK at the end
            if b[-33] in [0x02, 0x03]:
                return binascii.hexlify(b[-33:]).decode()
            # Check for uncompressed PK
            if len(b) >= 65 and b[-65] == 0x04:
                return binascii.hexlify(b[-65:]).decode()
    except: pass
    return None
with open('data/raw_txs_1JG64.json', 'r') as f:
    txs = json.load(f)
for tx in txs:
    for vin in tx.get('vin', []):
        if vin.get('prevout', {}).get('scriptpubkey_address') == '1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu':
            pk = extract_real_pk(vin.get('scriptsig', ''))
            if pk:
                print(f"FOUND PK for 1JG64: {pk}")
                break
