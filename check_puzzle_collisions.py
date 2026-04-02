import json

with open('data/puzzle_addresses.json', 'r') as f:
    puzzle_addrs = set(json.load(f))

with open('data/pubkey_collision_map.json', 'r') as f:
    collisions = json.load(f)

for pk, addrs in collisions.items():
    intersect = puzzle_addrs.intersection(set(addrs))
    if intersect:
        print(f"PK {pk} used by Puzzle Addrs: {list(intersect)}")
        if len(addrs) > len(intersect):
            others = set(addrs) - intersect
            print(f"  ...and other Addrs: {list(others)}")
