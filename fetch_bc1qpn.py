import requests
import json
def fetch_txs(address):
    url = f"https://mempool.space/api/address/{address}/txs"
    return requests.get(url).json()
txs = fetch_txs("bc1qpn52eagxlst0zyfa876rr9x8kz0cavxta4ym0h")
with open("data/raw_txs_bc1qpn.json", "w") as f:
    json.dump(txs, f)
