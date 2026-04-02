import requests
from collections import defaultdict
import json
import time
import binascii

def get_address_public_key(address):
    """
    Fetch the public key (if exposed) from the blockchain.
    A public key is exposed when an address has at least one outgoing transaction.
    """
    # Using blockchain.info rawaddr to see transactions
    url = f"https://blockchain.info/rawaddr/{address}"
    try:
        resp = requests.get(url, timeout=10)
        if resp.status_code != 200:
            return None
        data = resp.json()
        
        # We need to look at 'inputs' of transactions where this address was the sender
        for tx in data.get('txs', []):
            for inp in tx.get('inputs', []):
                prev_out = inp.get('prev_out', {})
                if prev_out.get('addr') == address:
                    script = inp.get('script', '')
                    if not script:
                        continue
                    
                    try:
                        # DER signatures usually look like: <sig_len><sig><pubkey_len><pubkey>
                        # Compressed pubkey is 33 bytes (66 hex chars), starts with 02 or 03
                        # Uncompressed is 65 bytes (130 hex chars), starts with 04
                        
                        # Heuristic: look for the end of the script where the pubkey usually sits
                        script_bytes = binascii.unhexlify(script)
                        
                        # Check for uncompressed (65 bytes)
                        if len(script_bytes) >= 65:
                            for i in range(len(script_bytes) - 65):
                                if script_bytes[i] == 0x04:
                                    return binascii.hexlify(script_bytes[i:i+65]).decode()
                        
                        # Check for compressed (33 bytes)
                        if len(script_bytes) >= 33:
                            for i in range(len(script_bytes) - 33):
                                if script_bytes[i] in [0x02, 0x03]:
                                    # Ensure it's not part of the signature (which usually ends in 0x01/SIGHASH_ALL)
                                    return binascii.hexlify(script_bytes[i:i+33]).decode()
                    except:
                        continue
        return None
    except:
        return None

def find_derivation_collisions(addresses, limit=100):
    """Find addresses that share a public key"""
    print(f"Scanning up to {limit} addresses for public key exposure...")
    pubkey_to_addrs = defaultdict(list)
    
    count = 0
    for addr in addresses:
        if count >= limit:
            break
            
        pubkey = get_address_public_key(addr)
        if pubkey:
            pubkey_to_addrs[pubkey].append(addr)
            print(f"Found pubkey for {addr}: {pubkey[:16]}...{pubkey[-16:]}")
        else:
            # Most dormant addresses never moved coins, so pubkey is NOT exposed
            # unless it was a P2PK (Pay to PubKey) address.
            pass
            
        count += 1
        time.sleep(0.5) # Avoid aggressive rate limiting
    
    # Find collisions
    collisions = {k: v for k, v in pubkey_to_addrs.items() if len(v) > 1}
    
    if collisions:
        print(f"\n[!] ALERT: DERIVATION COLLISIONS FOUND!")
        for pubkey, addrs in collisions.items():
            print(f"Pubkey: {pubkey}")
            for a in addrs:
                print(f"  - {a}")
    else:
        print("\nNo derivation collisions detected in this batch.")
        print(f"Total addresses with exposed pubkeys: {len(pubkey_to_addrs)}")
    
    return collisions

if __name__ == "__main__":
    import os
    list_path = 'bitcoin_crack/dormant_list.json'
    if os.path.exists(list_path):
        with open(list_path, 'r') as f:
            targets = json.load(f)
        
        # Filter for Legacy (starting with 1)
        legacy = [a for a in targets if a.startswith('1')]
        find_derivation_collisions(legacy, limit=50)
    else:
        print("Dormant list not found.")
