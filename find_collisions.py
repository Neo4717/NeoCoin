import json

with open('data/pubkey_collision_map.json', 'r') as f:
    data = json.load(f)

for pk, addrs in data.items():
    if len(addrs) > 1:
        print(f"Collision for PK {pk}:")
        for addr in addrs:
            print(f"  - {addr}")
