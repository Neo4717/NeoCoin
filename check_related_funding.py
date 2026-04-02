import requests
import json
def get_txs(address):
    url = f"https://mempool.space/api/address/{address}/txs"
    return requests.get(url).json()
def extract_pks(txs, address):
    pks = set()
    for tx in txs:
        for vin in tx.get('vin', []):
            if vin.get('prevout', {}).get('scriptpubkey_address') == address:
                # Extract PK from scriptsig
                ss = vin.get('scriptsig', '')
                if ss:
                    # Simple heuristic for P2PKH PK
                    b = bytes.fromhex(ss)
                    if len(b) >= 33:
                        if b[-33] in [0x02, 0x03]: pks.add(b[-33:].hex())
                        elif len(b) >= 65 and b[-65] == 0x04: pks.add(b[-65:].hex())
    return pks

addr1 = "1GNB9PdRPtc4R7cYyLTuUmkbBrfiXoW7Kp"
addr2 = "1PM8huQVFSirUT7eAwNm3rBBYTsDRzCaf3"

print(f"Checking {addr1}...")
pks1 = extract_pks(get_txs(addr1), addr1)
print(f"PKs for {addr1}: {pks1}")

print(f"Checking {addr2}...")
pks2 = extract_pks(get_txs(addr2), addr2)
print(f"PKs for {addr2}: {pks2}")
