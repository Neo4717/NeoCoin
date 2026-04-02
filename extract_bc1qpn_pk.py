import json
import binascii
def extract_witness_pk(witness):
    try:
        # For P2WPKH, witness is [sig, pubkey]
        if len(witness) == 2:
            pk = witness[1]
            if len(pk) == 66: # Compressed
                return pk
            if len(pk) == 130: # Uncompressed
                return pk
    except: pass
    return None
with open('data/raw_txs_bc1qpn.json', 'r') as f:
    txs = json.load(f)
for tx in txs:
    for vin in tx.get('vin', []):
        if vin.get('prevout', {}).get('scriptpubkey_address') == 'bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h':
            pk = extract_witness_pk(vin.get('witness', []))
            if pk:
                print(f"FOUND PK for bc1qpn: {pk}")
                break
