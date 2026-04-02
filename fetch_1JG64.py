import requests
import json
def fetch_txs(address):
    url = f"https://mempool.space/api/address/{address}/txs"
    return requests.get(url).json()
txs = fetch_txs("1JG648yaB7Wp2dpUfcZoRSD4q35oq47vCu")
with open("data/raw_txs_1JG64.json", "w") as f:
    json.dump(txs, f)
