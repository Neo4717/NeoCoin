import hashlib
import binascii
import json
import ecdsa

def privkey_to_p2pkh(priv_hex, compressed=True):
    try:
        priv_bytes = binascii.unhexlify(priv_hex)
        sk = ecdsa.SigningKey.from_string(priv_bytes, curve=ecdsa.SECP256k1)
        vk = sk.get_verifying_key()
        if compressed:
            prefix = b'\x02' if vk.to_string()[-1] % 2 == 0 else b'\x03'
            pubkey = prefix + vk.to_string()[:32]
        else:
            pubkey = b'\x04' + vk.to_string()
        
        h160 = hashlib.new('ripemd160', hashlib.sha256(pubkey).digest()).digest()
        # (Base58 encoding omitted for speed, we'll check H160)
        return h160.hex()
    except: return None

# Extract H160 from puzzle addresses
# (We can use the scriptpubkey_asm from data/raw_txs_1GSMG1.json)
# 76a914<h160>88ac

with open('data/puzzle_addresses.json', 'r') as f:
    puzzle_addrs = json.load(f)

# Collect all PKs we found
with open('data/puzzle_pks.json', 'r') as f:
    pk_map = json.load(f)

funders = {
    "145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ": "02d80a632a0d68c5114bed836a7866f9f94372a42ad833220ddcee727b48058db8",
    "1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu": "03b9ebca45d66cbc020feb99f3d3157d29ca302ab93dc81d2ff8551623a55c642e",
    "bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h": "0300856997e54cbb73ffc5c9148dd4af564289cb6d3a5e92d306207bcaccc96aef"
}

all_known_pks = set(pk_map.values())
for pk in funders.values(): all_known_pks.add(pk)

# We need a way to map H160 to Address
# I'll just use a simple list of H160s for now
target_h160s = set()
# ... (Need to extract them)

print(f"Checking {len(all_known_pks)} known PKs as SKs...")

for pk in all_known_pks:
    # Try the PK itself as a private key (skipping prefix if 33 bytes)
    if len(pk) == 66:
        privs = [pk[2:], pk]
    else:
        privs = [pk]
    
    for p in privs:
        if len(p) == 64:
            derived_h160_c = privkey_to_p2pkh(p, True)
            derived_h160_u = privkey_to_p2pkh(p, False)
            # Check if these H160s match ANY puzzle address
            # (I'll just print them for now and grep later)
            if derived_h160_c: print(f"PK_AS_SK:{p}:C:{derived_h160_c}")
            if derived_h160_u: print(f"PK_AS_SK:{p}:U:{derived_h160_u}")

