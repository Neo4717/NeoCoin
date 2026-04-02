import requests
import json
import binascii
import time

def scan_address_for_nonce_reuse(address):
    """
    Scans the transaction history of a given Bitcoin address for duplicate 'r' values.
    Uses the Blockchain.info API for fetching transaction data.
    """
    print(f"Scanning address: {address} for nonce reuse...")
    
    url = f"https://blockchain.info/rawaddr/{address}"
    max_retries = 3
    for attempt in range(max_retries):
        try:
            response = requests.get(url)
            if response.status_code == 200:
                break
            if response.status_code == 429:
                wait = (attempt + 1) * 10
                print(f"Rate limited (429). Waiting {wait}s...")
                time.sleep(wait)
                continue
            
            print(f"Error: API returned status code {response.status_code}")
            return None
        except Exception as e:
            print(f"Request error: {e}")
            time.sleep(2)
            continue
    else:
        print(f"Failed to fetch {address} after retries.")
        return None
            
    try:
        data = response.json()
        txs = data.get('tx', [])
        
        # r_value -> [(s_value, msg_hash, tx_id)]
        signatures = {}
        
        for tx in txs:
            tx_id = tx['hash']
            # Loop through inputs to find signatures
            for vin in tx.get('inputs', []):
                script = vin.get('script', '')
                if not script:
                    continue
                
                # Simple extraction of r and s from DER encoded signatures
                # DER format: 30 <len> 02 <len_r> <r> 02 <len_s> <s>
                try:
                    # In a real tool, we'd use a proper DER parser
                    # This is a simplified extraction for demonstration
                    script_bytes = binascii.unhexlify(script)
                    # Search for signature marker 0x30
                    idx = script_bytes.find(0x30)
                    if idx != -1:
                        # Skip 0x30 and length byte
                        sig = script_bytes[idx+2:]
                        # Parse r (starts with 02)
                        if len(sig) > 2 and sig[0] == 0x02:
                            len_r = sig[1]
                            if len(sig) >= 2 + len_r:
                                r_val = binascii.hexlify(sig[2:2+len_r]).decode()
                                
                                # For demonstration, we'll index them
                                if r_val in signatures:
                                    print(f"CRITICAL: Nonce reuse detected for r value: {r_val}")
                                    return True # Found reuse!
                                else:
                                    signatures[r_val] = tx_id
                except Exception as e:
                    # Skip non-standard scripts
                    continue
                    
        time.sleep(1) # Base delay to avoid 429
        return False

    except Exception as e:
        print(f"An error occurred: {e}")
        return None

if __name__ == "__main__":
    # Test with a known historical dormant address (or any address)
    target_address = "12t9YDPgwueZ9NyMgw519p7AA8isjr6SMw"
    scan_address_for_nonce_reuse(target_address)
