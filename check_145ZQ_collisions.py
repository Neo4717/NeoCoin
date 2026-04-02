import json
pk_145ZQ = "02d80a632a0d68c5114bed836a7866f9f94372a42ad833220ddcee727b48058db8"
with open('data/pubkey_collision_map.json', 'r') as f:
    collisions = json.load(f)
if pk_145ZQ in collisions:
    print(f"!!! 145ZQ PK COLLISION DETECTED !!!")
    print(f"  Addresses: {collisions[pk_145ZQ]}")
else:
    print("No collision found for 145ZQ in the map.")
