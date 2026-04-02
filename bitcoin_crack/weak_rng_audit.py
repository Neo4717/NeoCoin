import hashlib
import binascii
from ecdsa import SigningKey, SECP256k1
import json
import os

def get_address_from_privkey(priv_hex):
    """Derives a Legacy P2PKH address from a private key hex string."""
    sk = SigningKey.from_string(binascii.unhexlify(priv_hex), curve=SECP256k1)
    vk = sk.get_verifying_key()
    pubkey = b'\x04' + vk.to_string() # Uncompressed pubkey
    
    # SHA256 -> RIPEMD160
    sha256_hash = hashlib.sha256(pubkey).digest()
    ripemd160 = hashlib.new('ripemd160')
    ripemd160.update(sha256_hash)
    hash_160 = ripemd160.digest()
    
    # Add network byte (0x00 for Mainnet)
    network_hash = b'\x00' + hash_160
    
    # Double SHA256 for checksum
    checksum = hashlib.sha256(hashlib.sha256(network_hash).digest()).digest()[:4]
    
    # Base58 encode
    binary_addr = network_hash + checksum
    
    # Simple Base58 encoding
    alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
    num = int.from_bytes(binary_addr, byteorder='big')
    res = ""
    while num > 0:
        num, rem = divmod(num, 58)
        res = alphabet[rem] + res
        
    # Add leading '1's for zero bytes
    pad = 0
    for b in binary_addr:
        if b == 0: pad += 1
        else: break
        
    return "1" * pad + res

def audit_weak_seeds(target_addresses):
    """
    Simulates a weak RNG attack by generating private keys from low-entropy seeds
    and checking them against our dormant list.
    """
    print(f"Auditing {len(target_addresses)} addresses for weak RNG patterns...")
    
    # Common weak seed patterns (examples)
    weak_patterns = [
        "123456", "password", "bitcoin", "satoshi", "admin",
        "0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
        "qwerty", "test", "root", "freedom"
    ]
    
    found_any = False
    for pattern in weak_patterns:
        # Simulate SHA256 of the pattern as the private key
        # (This was common in 'Brain Wallets')
        seed_hash = hashlib.sha256(pattern.encode()).hexdigest()
        try:
            addr = get_address_from_privkey(seed_hash)
            if addr in target_addresses:
                print(f"!!! CRITICAL SUCCESS !!!")
                print(f"Pattern '{pattern}' matches dormant address: {addr}")
                print(f"Private Key (hex): {seed_hash}")
                found_any = True
        except Exception:
            continue
            
    if not found_any:
        print("No weak RNG matches found in the current pattern set.")

if __name__ == "__main__":
    list_path = 'bitcoin_crack/dormant_list.json'
    if os.path.exists(list_path):
        with open(list_path, 'r') as f:
            targets = json.load(f)
            audit_weak_seeds(targets)
    else:
        print("Dormant list not found. Run hunter/scraper first.")
