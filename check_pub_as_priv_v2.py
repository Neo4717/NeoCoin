import hashlib
import binascii
import json
import ecdsa

def privkey_to_pubkey(privkey_hex, compressed=True):
    try:
        privkey_bytes = binascii.unhexlify(privkey_hex)
        sk = ecdsa.SigningKey.from_string(privkey_bytes, curve=ecdsa.SECP256k1)
        vk = sk.get_verifying_key()
        if compressed:
            # prefix = 02 if y is even, 03 if y is odd
            prefix = b'\x02' if vk.to_string()[-1] % 2 == 0 else b'\x03'
            return binascii.hexlify(prefix + vk.to_string()[:32]).decode()
        else:
            return binascii.hexlify(b'\x04' + vk.to_string()).decode()
    except:
        return None

with open('data/puzzle_pks.json', 'r') as f:
    pk_map = json.load(f)

# Address -> PK
# We want PK -> Address
rev_pk_map = {pk: addr for addr, pk in pk_map.items()}
puzzle_pks = set(pk_map.values())

for addr, pk in pk_map.items():
    # Try using the PK itself (32 bytes from it) as a private key
    if len(pk) == 66:
        # 1. PK[2:] (skipping 02/03)
        priv_hex = pk[2:]
        derived_pk = privkey_to_pubkey(priv_hex, True)
        if derived_pk in puzzle_pks:
            print(f"!!! MATCH FOUND: {addr}'s PK as privkey leads to {rev_pk_map[derived_pk]}")

    # 2. sha256(PK)
    priv_hex_sha = hashlib.sha256(binascii.unhexlify(pk)).hexdigest()
    derived_pk_sha = privkey_to_pubkey(priv_hex_sha, True)
    if derived_pk_sha in puzzle_pks:
        print(f"!!! MATCH FOUND (SHA256): {addr}'s PK sha256 as privkey leads to {rev_pk_map[derived_pk_sha]}")

    # 3. PK itself as bytes (if it's 32 bytes uncompressed?)
    # ...

