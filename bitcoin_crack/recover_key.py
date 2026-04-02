import hashlib
from ecdsa import SECP256k1
import binascii

def recover_private_key(r, s1, z1, s2, z2):
    """
    Recovers the private key 'd' given two ECDSA signatures (r, s1) and (r, s2)
    for message hashes z1 and z2, sharing the same nonce 'k'.
    
    Formula:
    k = (z1 - z2) / (s1 - s2) mod n
    d = (s1*k - z1) / r mod n
    """
    n = SECP256k1.order
    
    # Calculate the modular inverse of (s1 - s2)
    s_diff = (s1 - s2) % n
    if s_diff == 0:
        return None  # Cannot recover if s1 == s2
        
    s_diff_inv = pow(s_diff, -1, n)
    
    # Recover k (the nonce)
    k = ((z1 - z2) * s_diff_inv) % n
    
    # Calculate the modular inverse of r
    r_inv = pow(r, -1, n)
    
    # Recover d (the private key)
    d = ((s1 * k - z1) * r_inv) % n
    
    return d

def test_recovery():
    # Example values (mocked for testing logic)
    # In a real scenario, these would be extracted from the blockchain
    r = 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
    s1 = 0x1
    z1 = 0x2
    s2 = 0x3
    z2 = 0x4
    
    # This is a dummy test, real vectors would be needed for a meaningful check
    d = recover_private_key(r, s1, z1, s2, z2)
    if d:
        print(f"Recovered private key (hex): {hex(d)}")
    else:
        print("Failed to recover key.")

if __name__ == "__main__":
    test_recovery()
