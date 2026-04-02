import requests
import json
import time
import os

def get_pubkey(address):
    """
    Attempts to find the public key for a given address.
    Checks spending transactions (scriptsig) and P2PK receiving transactions.
    """
    url = f"https://mempool.space/api/address/{address}/txs"
    try:
        response = requests.get(url, timeout=10)
        if response.status_code != 200:
            return None
        txs = response.json()
        
        # 1. Check spending transactions (vin)
        for tx in txs:
            for vin in tx.get('vin', []):
                prevout = vin.get('prevout')
                if prevout and prevout.get('scriptpubkey_address') == address:
                    scriptsig_asm = vin.get('scriptsig_asm', '')
                    parts = scriptsig_asm.split(' ')
                    for part in parts:
                        # Pubkeys are 33 bytes (66 chars) or 65 bytes (130 chars)
                        if len(part) in [66, 130] and part.startswith(('02', '03', '04')):
                            return part
            
            # 2. Check if it was a P2PK output (receiving)
            for vout in tx.get('vout', []):
                if vout.get('scriptpubkey_address') == address:
                    asm = vout.get('scriptpubkey_asm', '')
                    # P2PK: <pubkey> OP_CHECKSIG
                    if 'OP_CHECKSIG' in asm and 'OP_HASH160' not in asm:
                        parts = asm.split(' ')
                        for part in parts:
                            if len(part) in [66, 130] and part.startswith(('02', '03', '04')):
                                return part
        return None
    except Exception as e:
        print(f"Error fetching {address}: {e}")
        return None

def main():
    input_file = 'bitcoin_crack/dormant_list.json'
    output_file = 'data/pubkey_map.json'
    
    if not os.path.exists('data'):
        os.makedirs('data')
        
    with open(input_file, 'r') as f:
        all_addresses = json.load(f)
        
    # Filter for legacy '1...' addresses and take the first 50
    legacy_addresses = [addr for addr in all_addresses if addr.startswith('1')][:50]
    
    print(f"Starting hunter for {len(legacy_addresses)} legacy addresses...")
    
    pubkey_map = {}
    
    for i, addr in enumerate(legacy_addresses):
        print(f"[{i+1}/{len(legacy_addresses)}] Checking {addr}...", end='', flush=True)
        pubkey = get_pubkey(addr)
        if pubkey:
            print(f" FOUND: {pubkey[:10]}...{pubkey[-10:]}")
            pubkey_map[addr] = pubkey
        else:
            print(" NOT FOUND")
        
        # Rate limiting
        time.sleep(0.5)
        
    # Save mapping
    with open(output_file, 'w') as f:
        json.dump(pubkey_map, f, indent=4)
    
    print(f"\nSaved {len(pubkey_map)} mappings to {output_file}")
    
    # Identify collisions
    rev_map = {}
    collisions = []
    for addr, pk in pubkey_map.items():
        if pk in rev_map:
            rev_map[pk].append(addr)
        else:
            rev_map[pk] = [addr]
            
    for pk, addrs in rev_map.items():
        if len(addrs) > 1:
            collisions.append({
                "pubkey": pk,
                "addresses": addrs
            })
            
    if collisions:
        print("\n!!! COLLISIONS FOUND !!!")
        for c in collisions:
            print(f"PubKey: {c['pubkey']}")
            print(f"Addresses: {', '.join(c['addresses'])}")
    else:
        print("\nNo pubkey collisions found among the processed addresses.")

if __name__ == "__main__":
    main()
