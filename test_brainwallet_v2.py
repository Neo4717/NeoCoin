import hashlib
import binascii
import ecdsa

# Simplified bech32 encode for matching
CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
def bech32_polymod(values):
    generator = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]
    chk = 1
    for value in values:
        top = chk >> 25
        chk = (chk & 0x1ffffff) << 5 ^ value
        for i in range(5):
            chk ^= generator[i] if ((top >> i) & 1) else 0
    return chk
def bech32_hrp_expand(hrp):
    return [ord(x) >> 5 for x in hrp] + [0] + [ord(x) & 31 for x in hrp]
def bech32_create_checksum(hrp, data, spec):
    values = bech32_hrp_expand(hrp) + data
    const = 0x2f43684b if spec == 2 else 1
    polymod = bech32_polymod(values + [0, 0, 0, 0, 0, 0]) ^ const
    return [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]
def bech32_encode(hrp, data, spec):
    combined = data + bech32_create_checksum(hrp, data, spec)
    return hrp + '1' + ''.join([CHARSET[d] for d in combined])
def convertbits(data, frombits, tobits, pad=True):
    acc = 0
    bits = 0
    ret = []
    maxv = (1 << tobits) - 1
    max_acc = (1 << (frombits + tobits - 1)) - 1
    for value in data:
        if value < 0 or (value >> frombits): return None
        acc = ((acc << frombits) | value) & max_acc
        bits += frombits
        while bits >= tobits:
            bits -= tobits
            ret.append((acc >> bits) & maxv)
    if pad:
        if bits:
            ret.append((acc << (tobits - bits)) & maxv)
    return ret
def privkey_to_address(priv_hex):
    priv_bytes = binascii.unhexlify(priv_hex)
    sk = ecdsa.SigningKey.from_string(priv_bytes, curve=ecdsa.SECP256k1)
    vk = sk.get_verifying_key()
    prefix = b'\x02' if vk.to_string()[-1] % 2 == 0 else b'\x03'
    pubkey = prefix + vk.to_string()[:32]
    sha256_h = hashlib.sha256(pubkey).digest()
    ripemd160_h = hashlib.new('ripemd160', sha256_h).digest()
    witver = 0
    witprog = list(ripemd160_h)
    data = [witver] + convertbits(witprog, 8, 5)
    return bech32_encode('bc', data, 1)

passphrases = [
    "women",
    "woman",
    "Trinity",
    "The Oracle",
    "Oracle",
    "Neo",
    "Morpheus",
    "The Matrix",
    "Matrix",
    "There is no spoon",
    "ALPHANOISES",
    "SalPhaseIon",
    "Everything That Has A Beginning Has An End",
    "HALVING",
    "Turing Complete",
    "Causality Transcended"
]

target = "bc1qks8zrshwmu3m8vgqdzwl2u8jjfgnvgjlezwqcd"

for p in passphrases:
    priv_hex = hashlib.sha256(p.encode()).hexdigest()
    derived = privkey_to_address(priv_hex)
    if derived == target:
        print(f"!!! MATCH FOUND !!! Pass: {p} -> {derived}")
    # Also try lowercase
    priv_hex_l = hashlib.sha256(p.lower().encode()).hexdigest()
    derived_l = privkey_to_address(priv_hex_l)
    if derived_l == target:
        print(f"!!! MATCH FOUND !!! Pass: {p.lower()} -> {derived_l}")

