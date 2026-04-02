import hashlib
import binascii
import base58
import ecdsa

def pubkey_to_address(pubkey_hex):
    pubkey_bytes = binascii.unhexlify(pubkey_hex)
    sha256_h = hashlib.sha256(pubkey_bytes).digest()
    ripemd160_h = hashlib.new('ripemd160', sha256_h).digest()
    v_h = b'\x00' + ripemd160_h
    checksum = hashlib.sha256(hashlib.sha256(v_h).digest()).digest()[:4]
    return base58.b58encode(v_h + checksum).decode()

def privkey_to_pubkey(privkey_hex, compressed=True):
    privkey_bytes = binascii.unhexlify(privkey_hex)
    sk = ecdsa.SigningKey.from_string(privkey_bytes, curve=ecdsa.SECP256k1)
    vk = sk.get_verifying_key()
    if compressed:
        prefix = b'\x02' if vk.to_string()[-1] % 2 == 0 else b'\x03'
        return binascii.hexlify(prefix + vk.to_string()[:32]).decode()
    else:
        return binascii.hexlify(b'\x04' + vk.to_string()).decode()

with open('data/puzzle_pks.json', 'r') as f:
    pk_map = json.load(f)

puzzle_addresses = set(pk_map.keys())

for addr, pk in pk_map.items():
    # Try using the PK itself (32 bytes from it) as a private key
    # Most compressed PKs are 33 bytes: 02/03 + 32 bytes
    if len(pk) == 66:
        priv_hex = pk[2:]
        try:
            derived_pub_c = privkey_to_pubkey(priv_hex, True)
            derived_addr_c = pubkey_to_address(derived_pub_c)
            if derived_addr_c in puzzle_addresses:
                print(f"!!! MATCH FOUND: {addr}'s PK as privkey leads to {derived_addr_c}")
        except:
            pass

    # Try using the whole PK hex as a string/passphrase?
    # Or try sha256(PK) as privkey
    priv_hex_sha = hashlib.sha256(binascii.unhexlify(pk)).hexdigest()
    derived_pub_sha = privkey_to_pubkey(priv_hex_sha, True)
    derived_addr_sha = pubkey_to_address(derived_pub_sha)
    if derived_addr_sha in puzzle_addresses:
        print(f"!!! MATCH FOUND (SHA256): {addr}'s PK sha256 as privkey leads to {derived_addr_sha}")

