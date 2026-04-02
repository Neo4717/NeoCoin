import json
import binascii

def extract_r_s(sig_hex):
    try:
        b = binascii.unhexlify(sig_hex)
        if b[0] == 0x30:
            r_len = b[3]
            r = b[4:4+r_len]
            if len(r) > 0 and r[0] == 0x00: r = r[1:]
            s_len = b[4+r_len+1]
            s = b[4+r_len+2:4+r_len+2+s_len]
            if len(s) > 0 and s[0] == 0x00: s = s[1:]
            return r.hex(), s.hex()
    except: pass
    return None, None

with open('data/full_audit_1GSMG1.json', 'r') as f:
    data = json.load(f)

r_map = {} # r -> list of (txid, s)

for item in data:
    for vin in item.get('vin', []):
        ss = vin.get('scriptsig')
        if ss:
            # ScriptSig format can vary, but let's try to extract sig
            # Usually <sig_len><sig><pubkey_len><pubkey>
            # Sig starts with 30
            try:
                b = binascii.unhexlify(ss)
                idx = b.find(b'\x30')
                if idx != -1:
                    # Found a potential signature
                    r, s = extract_r_s(ss[idx*2:])
                    if r:
                        if r in r_map:
                            for prev_txid, prev_s in r_map[r]:
                                if prev_s != s:
                                    print(f"!!! VULNERABLE NONCE REUSE !!!")
                                    print(f"  R: {r}")
                                    print(f"  TX1: {prev_txid} (S: {prev_s})")
                                    print(f"  TX2: {item['txid']} (S: {s})")
                            r_map[r].append((item['txid'], s))
                        else:
                            r_map[r] = [(item['txid'], s)]
            except: pass

print("Audit complete.")
