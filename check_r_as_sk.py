import hashlib
import binascii
import json
import ecdsa

def h160(pubkey_bytes):
    return hashlib.new('ripemd160', hashlib.sha256(pubkey_bytes).digest()).digest().hex()

def priv_to_h160s(priv_hex):
    try:
        priv_bytes = binascii.unhexlify(priv_hex)
        sk = ecdsa.SigningKey.from_string(priv_bytes, curve=ecdsa.SECP256k1)
        vk = sk.get_verifying_key()
        pk_c = (b'\x02' if vk.to_string()[-1] % 2 == 0 else b'\x03') + vk.to_string()[:32]
        pk_u = b'\x04' + vk.to_string()
        return h160(pk_c), h160(pk_u)
    except: return None, None

with open('blockchain_h160s.txt', 'r') as f:
    target_h160s = set(line.strip() for line in f)

# Extract R values from full_audit_1GSMG1.json
with open('data/full_audit_1GSMG1.json', 'r') as f:
    audit_data = json.load(f)

print(f"Checking {len(audit_data)} transaction signatures for R-as-SK...")

for item in audit_data:
    for vin in item.get('vin', []):
        ss = vin.get('scriptsig')
        if ss:
            try:
                b = binascii.unhexlify(ss)
                idx = b.find(b'\x30')
                if idx != -1:
                    sig = b[idx:]
                    r_len = sig[3]
                    r = sig[4:4+r_len]
                    if len(r) > 0 and r[0] == 0x00: r = r[1:]
                    if len(r) == 32:
                        r_hex = r.hex()
                        h_c, h_u = priv_to_h160s(r_hex)
                        if h_c in target_h160s: print(f"!!! MATCH FOUND (R-as-SK, Compressed) !!! R: {r_hex} leads to H160: {h_c}")
                        if h_u in target_h160s: print(f"!!! MATCH FOUND (R-as-SK, Uncompressed) !!! R: {r_hex} leads to H160: {h_u}")
            except: pass

print("Audit finished.")
