import json
import os
import time
import concurrent.futures
import threading
import requests
import binascii

# Import the existing scanner to reuse logic if possible, 
# but we'll reimplement the core to ensure exact compliance with requirements.
import scanner

LOCK = threading.Lock()
LAST_REQUEST_TIME = 0

def scan_with_compliance(address):
    global LAST_REQUEST_TIME
    
    # 4. Use 2 parallel workers and maintain a 2-second delay between requests to avoid 429 errors.
    with LOCK:
        now = time.time()
        elapsed = now - LAST_REQUEST_TIME
        if elapsed < 2:
            time.sleep(2 - elapsed)
        LAST_REQUEST_TIME = time.time()

    url = f"https://blockchain.info/rawaddr/{address}"
    
    while True:
        try:
            response = requests.get(url, timeout=30)
            if response.status_code == 200:
                break
            if response.status_code == 429:
                # 5. If a 429 error occurs, wait 30 seconds before retrying.
                print(f"Rate limited (429) for {address}. Waiting 30s...")
                time.sleep(30)
                continue
            
            print(f"Error {response.status_code} for {address}")
            return None
        except Exception as e:
            print(f"Request error for {address}: {e}")
            time.sleep(5)
            continue

    try:
        data = response.json()
        txs = data.get('tx', [])
        signatures = {}
        for tx in txs:
            tx_id = tx['hash']
            for vin in tx.get('inputs', []):
                script = vin.get('script', '')
                if not script: continue
                try:
                    script_bytes = binascii.unhexlify(script)
                    idx = script_bytes.find(0x30)
                    if idx != -1:
                        sig = script_bytes[idx+2:]
                        if len(sig) > 2 and sig[0] == 0x02:
                            len_r = sig[1]
                            if len(sig) >= 2 + len_r:
                                r_val = binascii.hexlify(sig[2:2+len_r]).decode()
                                if r_val in signatures:
                                    return True
                                else:
                                    signatures[r_val] = tx_id
                except:
                    continue
        return False
    except Exception as e:
        print(f"JSON/Parsing error for {address}: {e}")
        return None

def main():
    list_path = 'bitcoin_crack/dormant_list.json'
    results_path = 'bitcoin_crack/audit_results.json'

    # 1. Read the existing results
    if os.path.exists(results_path):
        with open(results_path, 'r') as f:
            results = json.load(f)
    else:
        results = {}

    with open(list_path, 'r') as f:
        all_addresses = json.load(f)

    # 2. Identify which addresses still have 'null' or are missing from the results.
    todo = [addr for addr in all_addresses if addr not in results or results[addr] is None]
    
    print(f"Total addresses: {len(all_addresses)}")
    print(f"Already scanned: {len(all_addresses) - len(todo)}")
    print(f"Remaining to scan: {len(todo)}")
    if todo:
        print(f"First 5 todo: {todo[:5]}")

    if not todo:
        print("All addresses already scanned.")
        return

    newly_scanned = 0
    # 4. Use 2 parallel workers
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
        future_to_addr = {executor.submit(scan_with_compliance, addr): addr for addr in todo}
        
        try:
            for future in concurrent.futures.as_completed(future_to_addr):
                addr = future_to_addr[future]
                try:
                    res = future.result()
                    results[addr] = res
                    newly_scanned += 1
                    print(f"[{newly_scanned}/{len(todo)}] Scanned {addr}: {res}")
                    
                    # 6. Update the JSON results file with the new data (incrementally to be safe)
                    with open(results_path, 'w') as f:
                        json.dump(results, f, indent=4)
                        
                except Exception as e:
                    print(f"Failed to scan {addr}: {e}")
        except KeyboardInterrupt:
            print("Scan interrupted by user.")

    print(f"Scan session complete. Newly scanned: {newly_scanned}")

if __name__ == "__main__":
    main()
