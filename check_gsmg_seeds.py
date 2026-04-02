import hashlib
import binascii
import ecdsa

# Bech32 stuff
CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
def bech32_polymod(values):
    generator = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]
    chk = 1
    for value in values:
        top = chk >> 25
        chk = (chk & 0x1ffffff) << 5 ^ value
        for i in range(5): chk ^= generator[i] if ((top >> i) & 1) else 0
    return chk
def bech32_hrp_expand(hrp): return [ord(x) >> 5 for x in hrp] + [0] + [ord(x) & 31 for x in hrp]
def bech32_create_checksum(hrp, data, spec):
    values = bech32_hrp_expand(hrp) + data
    const = 0x2f43684b if spec == 2 else 1
    polymod = bech32_polymod(values + [0, 0, 0, 0, 0, 0]) ^ const
    return [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]
def bech32_encode(hrp, data, spec):
    combined = data + bech32_create_checksum(hrp, data, spec)
    return hrp + '1' + ''.join([CHARSET[d] for d in combined])
def convertbits(data, frombits, tobits, pad=True):
    acc = 0; bits = 0; ret = []; maxv = (1 << tobits) - 1; max_acc = (1 << (frombits + tobits - 1)) - 1
    for value in data:
        if value < 0 or (value >> frombits): return None
        acc = ((acc << frombits) | value) & max_acc; bits += frombits
        while bits >= tobits: bits -= tobits; ret.append((acc >> bits) & maxv)
    if pad:
        if bits: ret.append((acc << (tobits - bits)) & maxv)
    return ret

def get_addr(priv_hex, segwit=True):
    try:
        priv_bytes = binascii.unhexlify(priv_hex)
        sk = ecdsa.SigningKey.from_string(priv_bytes, curve=ecdsa.SECP256k1)
        vk = sk.get_verifying_key()
        
        # Compressed
        prefix_c = b'\x02' if vk.to_string()[-1] % 2 == 0 else b'\x03'
        pubkey_c = prefix_c + vk.to_string()[:32]
        h160_c = hashlib.new('ripemd160', hashlib.sha256(pubkey_c).digest()).digest()
        
        if segwit:
            data = [0] + convertbits(list(h160_c), 8, 5)
            return bech32_encode('bc', data, 1)
        else:
            # P2PKH Compressed
            network_hash = b'\x00' + h160_c
            checksum = hashlib.sha256(hashlib.sha256(network_hash).digest()).digest()[:4]
            import base58
            return base58.b58encode(network_hash + checksum).decode()
    except: return None

seeds = [
    "HALF", "BETTER HALF", "half", "better half",
    "causality", "theseedisplanted", "HASHTHETEXT",
    "itisonlywiththeheartthatoneseesrightlywhatisessentialisinvisibletotheeye",
    "ALPHANOISES", "SalPhaseIon", "THEMATRIXHASYOU", "There is no spoon",
    "women", "Trinity", "Neo", "Morpheus", "Architect", "choice",
    "673b7b4b67571b1b4b-3.o", "844e86a69a04eea672049e0e0e8612"
]

target_bech32 = "bc1qks8zrshwmu3m8vgqdzwl2u8jjfgnvgjlezwqcd"
target_prize = "1GSMG1JC9wtdSwfwApgj2xcmJPAwx7prBe"

for s in seeds:
    priv = hashlib.sha256(s.encode()).hexdigest()
    addr_b = get_addr(priv, True)
    addr_p = get_addr(priv, False)
    if addr_b == target_bech32: print(f"MATCH (Bech32): {s} -> {priv}")
    if addr_p == target_prize: print(f"MATCH (Prize): {s} -> {priv}")

