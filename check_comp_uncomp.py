import json
import hashlib
import binascii
import ecdsa

def get_uncomp(compressed_pk):
    try:
        pk_bytes = binascii.unhexlify(compressed_pk)
        sk = ecdsa.VerifyingKey.from_string(pk_bytes[1:], curve=ecdsa.SECP256k1, validate_point=False)
        # This isn't quite right, we need to recover the y coordinate
        # Use ecdsa's point recovery
        # ... actually easier to just use the VerifyingKey from point
        pass
    except: return None

with open('data/pubkey_collision_map.json', 'r') as f:
    collisions = json.load(f)

# Collect all PKs
all_pks = set(collisions.keys())

# This is hard to do without full point recovery. 
# Let's try a different approach: check if any H160 is shared? 
# No, H160 depends on the PK bytes.

print("Checking for PKs that might be compressed/uncompressed versions of each other...")
# ...
