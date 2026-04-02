import json
import concurrent.futures
import scanner
import os

def perform_audit(max_workers=2):
    list_path = 'bitcoin_crack/dormant_list.json'
    results_path = 'bitcoin_crack/audit_results.json'
    
    if not os.path.exists(list_path):
        print(f"Error: {list_path} not found.")
        return
        
    with open(list_path, 'r') as f:
        data = json.load(f)
        # Ensure it's a list of address strings
        if isinstance(data, list):
            addresses = data
        elif isinstance(data, dict):
            # If it's a dict, try to find a list in it
            for key, val in data.items():
                if isinstance(val, list):
                    addresses = val
                    break
            else:
                print("Error: Could not find address list in JSON.")
                return
    
    print(f"Starting parallel scan of {len(addresses)} addresses using {max_workers} workers...")
    
    results = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        # Pass the function and the list of addresses
        future_to_addr = {executor.submit(scanner.scan_address_for_nonce_reuse, addr): addr for addr in addresses}
        
        for future in concurrent.futures.as_completed(future_to_addr):
            addr = future_to_addr[future]
            try:
                results[addr] = future.result()
                if results[addr]:
                    print(f"!!! ALERT: Nonce reuse potential found at {addr} !!!")
            except Exception as e:
                print(f"Error scanning {addr}: {e}")
                results[addr] = f"Error: {e}"
            
    with open(results_path, 'w') as f:
        json.dump(results, f, indent=4)
    
    print(f"Scan complete. Results saved to {results_path}")

if __name__ == "__main__":
    perform_audit()
