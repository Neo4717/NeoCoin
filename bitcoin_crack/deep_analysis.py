import json
import binascii
from collections import defaultdict

def deep_analyze(file_path):
    print(f"--- Deep Signature Analysis: {file_path} ---")
    with open(file_path, 'r') as f:
        txs = json.load(f)

    # Maps to track potential leaks
    r_map = defaultdict(list)
    s_map = defaultdict(list)
    pubkey_map = defaultdict(list)

    for tx in txs:
        txid = tx['txid']
        for vin in tx.get('vin', []):
            script_hex = vin.get('scriptsig', '')
            if not script_hex:
                continue

            try:
                script = binascii.unhexlify(script_hex)
                # Parse DER Signature and PubKey
                # 0x30 <len> 0x02 <r_len> <r> 0x02 <s_len> <s> <sighash> <pubkey_push> <pubkey>
                
                # Simple extraction for analysis
                for i in range(len(script) - 10):
                    if script[i] == 0x30:
                        r_len = script[i+3]
                        r_val = binascii.hexlify(script[i+4:i+4+r_len]).decode()
                        
                        s_marker = i+4+r_len
                        if script[s_marker] == 0x02:
                            s_len = script[s_marker+1]
                            s_val = binascii.hexlify(script[s_marker+2:s_marker+2+s_len]).decode()
                            
                            r_map[r_val].append(txid)
                            s_map[s_val].append(txid)
                            
                            # Pubkey usually follows the signature
                            # (Heuristic: look for 0x21 or 0x41 after the sig)
                            pk_marker = s_marker+2+s_len+1
                            if pk_marker < len(script):
                                pk_len = script[pk_marker]
                                if pk_len in [33, 65] and pk_marker+1+pk_len <= len(script):
                                    pubkey = binascii.hexlify(script[pk_marker+1:pk_marker+1+pk_len]).decode()
                                    pubkey_map[pubkey].append(txid)
            except:
                continue

    print(f"Transactions Scanned: {len(txs)}")
    
    # 1. Check for Duplicate R (Nonce Reuse)
    r_collisions = {r: txids for r, txids in r_map.items() if len(txids) > 1}
    if r_collisions:
        print(f"\n[!] ALERT: Duplicate R-Values found!")
        for r, ids in r_collisions.items():
            print(f"R: {r} in TXs: {ids}")
    else:
        print("\n[✓] No duplicate R-values (nonce reuse) found.")

    # 2. Check for Duplicate S (Signer pattern)
    s_collisions = {s: txids for s, txids in s_map.items() if len(txids) > 1}
    if s_collisions:
        print(f"\n[!] ALERT: Duplicate S-Values found!")
        for s, ids in s_collisions.items():
            print(f"S: {s} in TXs: {ids}")
    else:
        print("[✓] No duplicate S-values found.")

    # 3. Check for Public Key Reuse
    pk_usage = {pk: txids for pk, txids in pubkey_map.items() if len(set(txids)) > 1}
    if pk_usage:
        print(f"\n[i] Public Key Reuse detected (Normal for wallets):")
        for pk, ids in list(pk_usage.items())[:3]: # Limit output
            print(f"PubKey: {pk[:32]}... used in {len(ids)} transactions")
    
    # 4. Check for "Symmetry" or "Small Value" leaks
    for r, ids in r_map.items():
        if int(r, 16) < 1000000:
            print(f"\n[!] WARNING: Abnormally small R-value found: {r} in {ids}")
    for s, ids in s_map.items():
        if int(s, 16) < 1000000:
            print(f"\n[!] WARNING: Abnormally small S-value found: {s} in {ids}")

if __name__ == "__main__":
    deep_analyze('data/full_audit_1GSMG1.json')
