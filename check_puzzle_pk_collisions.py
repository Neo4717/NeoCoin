import json

with open('data/puzzle_pks.json', 'r') as f:
    pk_map = json.load(f)

rev_map = {}
for addr, pk in pk_map.items():
    if pk in rev_map:
        rev_map[pk].append(addr)
    else:
        rev_map[pk] = [addr]

for pk, addrs in rev_map.items():
    if len(addrs) > 1:
        print(f"!!! COLLISION IN PUZZLE ADDRESSES !!!")
        print(f"  Shared PK: {pk}")
        print(f"  Addresses: {addrs}")
