import json
import binascii

def decode_hex(hex_str):
    try:
        return binascii.unhexlify(hex_str).decode('ascii', errors='replace')
    except:
        return None

with open('data/raw_txs_1GSMG1.json', 'r') as f:
    txs = json.load(f)

# Sort by block height then by time
txs.sort(key=lambda x: (x.get('status', {}).get('block_height', 0), x.get('status', {}).get('block_time', 0)))

for tx in txs:
    for vout in tx.get('vout', []):
        asm = vout.get('scriptpubkey_asm', '')
        if 'OP_RETURN' in asm:
            hex_data = asm.split(' ')[-1]
            decoded = decode_hex(hex_data)
            if decoded:
                print(f"[{tx['status']['block_height']}] {decoded}")
