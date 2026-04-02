import json
import binascii
import hashlib

def compress_pk(uncomp_pk):
    try:
        b = binascii.unhexlify(uncomp_pk)
        if b[0] != 0x04: return None
        x = b[1:33]
        y = b[33:]
        prefix = b'\x02' if y[-1] % 2 == 0 else b'\x03'
        return (prefix + x).hex()
    except: return None

with open('data/pubkey_map.json', 'r') as f:
    uncomp_map = json.load(f)

with open('data/pubkey_collision_map.json', 'r') as f:
    comp_map = json.load(f)

for u_addr, u_pk in uncomp_map.items():
    c_pk = compress_pk(u_pk)
    if c_pk and c_pk in comp_map:
        print(f"!!! MIXED COMPRESSION COLLISION !!!")
        print(f"  Uncompressed Addr: {u_addr} (PK: {u_pk[:16]}...)")
        print(f"  Compressed Addrs:   {comp_map[c_pk]} (PK: {c_pk})")

