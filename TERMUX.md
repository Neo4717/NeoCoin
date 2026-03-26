# NeoCoin on Termux (Android)

## Quick Install

```bash
# 1. Install Go and Git in Termux
pkg update
pkg install golang git

# 2. Clone NeoCoin
cd ~
git clone https://github.com/Neo4717/NeoCoin.git
cd NeoCoin/blockchain

# 3. Build
go build -o neocoin .

# 4. Create wallet
./neocoin create_wallet
# Save your address!

# 5. Run node
# Solo (no peer):
MINER_ADDRESS=YOUR_ADDRESS GENESIS_PATH=../genesis/smoke.json CHAIN_ID=3 AUTO_MINE=true MINE_FORCE_EMPTY_BLOCKS=true ./neocoin server

# With peer (replace PEER_IP with your computer's IP):
MINER_ADDRESS=YOUR_ADDRESS P2P_PEERS=PEER_IP:9090 GENESIS_PATH=../genesis/smoke.json CHAIN_ID=3 AUTO_MINE=true MINE_FORCE_EMPTY_BLOCKS=true ./neocoin server
```

## Find Your Computer IP (on computer)

```bash
# Linux/Mac:
ip addr show | grep inet

# Windows:
ipconfig
```

## Check Node

```bash
curl http://127.0.0.1:8080/chain/info
```

## Get Your Peer Address

Share this from your computer:
```
P2P Port: 9090
API Port: 8080
```
