import json
import hashlib
import binascii
import ecdsa

def privkey_to_p2pkh(priv_hex):
    try:
        priv_bytes = binascii.unhexlify(priv_hex)
        sk = ecdsa.SigningKey.from_string(priv_bytes, curve=ecdsa.SECP256k1)
        vk = sk.get_verifying_key()
        
        # Compressed
        prefix_c = b'\x02' if vk.to_string()[-1] % 2 == 0 else b'\x03'
        pubkey_c = prefix_c + vk.to_string()[:32]
        h160_c = hashlib.new('ripemd160', hashlib.sha256(pubkey_c).digest()).digest()
        
        # Uncompressed
        pubkey_u = b'\x04' + vk.to_string()
        h160_u = hashlib.new('ripemd160', hashlib.sha256(pubkey_u).digest()).digest()
        
        return binascii.hexlify(h160_c).decode(), binascii.hexlify(h160_u).decode()
    except: return None, None

def address_to_h160(address):
    # Simple decode for standard 1... addresses
    import base58
    try:
        decoded = base58.b58decode_check(address)
        return decoded[1:].hex()
    except: return None

prize_address = "1GSMG1JC9wtdSwfwApgj2xcmJPAwx7prBe"
prize_h160 = "a9553269572a317e39f0f518cb87c1a0ee1dbae4" # Extracted from grep earlier

with open('data/pubkey_collision_map.json', 'r') as f:
    collisions = json.load(f)

for pk in collisions.keys():
    # PK is 33 or 65 bytes
    # Try the PK itself as a private key (skipping prefix if 33 bytes)
    if len(pk) == 66:
        privs = [pk[2:], pk]
    else:
        privs = [pk]
        
    for p in privs:
        if len(p) == 64:
            h160_c, h160_u = privkey_to_p2pkh(p)
            if h160_c == prize_h160 or h160_u == prize_h160:
                print(f"!!! MATCH FOUND !!! Private key: {p}")

