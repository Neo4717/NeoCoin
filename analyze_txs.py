import json
import binascii

def decode_hex(hex_str):
    try:
        return binascii.unhexlify(hex_str).decode('ascii', errors='replace')
    except:
        return None

with open('data/raw_txs_1GSMG1.json', 'r') as f:
    txs = json.load(f)

for tx in txs:
    print(f"TXID: {tx['txid']}")
    for vin in tx.get('vin', []):
        ss = vin.get('scriptsig_asm', '')
        if ss:
            print(f"  ScriptSig: {ss}")
        w = vin.get('witness', [])
        if w:
            for item in w:
                decoded = decode_hex(item)
                if decoded and any(c.isprintable() for c in decoded):
                    print(f"  Witness (decoded): {decoded}")
    for vout in tx.get('vout', []):
        asm = vout.get('scriptpubkey_asm', '')
        if 'OP_RETURN' in asm:
            print(f"  Vout (OP_RETURN): {asm}")
            hex_data = asm.split(' ')[-1]
            decoded = decode_hex(hex_data)
            if decoded:
                print(f"    Decoded: {decoded}")
