import json

with open('data/puzzle_pks.json', 'r') as f:
    puzzle_pks = json.load(f)

funders = {
    "145ZQ9siLrsXBKf465wjdyQYAP5dRwhRhQ": "02d80a632a0d68c5114bed836a7866f9f94372a42ad833220ddcee727b48058db8",
    "1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu": "03b9ebca45d66cbc020feb99f3d3157d29ca302ab93dc81d2ff8551623a55c642e",
    "bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h": "0300856997e54cbb73ffc5c9148dd4af564289cb6d3a5e92d306207bcaccc96aef"
}

for p_addr, p_pk in puzzle_pks.items():
    for f_addr, f_pk in funders.items():
        if p_pk == f_pk:
            print(f"!!! MATCH FOUND !!!")
            print(f"  Puzzle Address: {p_addr}")
            print(f"  Funder Address: {f_addr}")
            print(f"  Shared PubKey:  {p_pk}")

