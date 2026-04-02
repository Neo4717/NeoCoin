package scanner

import (
	"encoding/hex"
)

// ExtractPubKey attempts to find a compressed (33b) or uncompressed (65b)
// public key within a scriptSig.
func ExtractPubKey(scriptHex string) string {
	script, err := hex.DecodeString(scriptHex)
	if err != nil {
		return ""
	}

	// Heuristic: Look for common pubkey markers
	for i := 0; i < len(script); i++ {
		// Uncompressed: 0x04 followed by 64 bytes
		if script[i] == 0x04 && i+64 < len(script) {
			return hex.EncodeToString(script[i : i+65])
		}
		// Compressed: 0x02 or 0x03 followed by 32 bytes
		if (script[i] == 0x02 || script[i] == 0x03) && i+32 < len(script) {
			// Ensure it's not a small push or part of a signature
			// (Simple heuristic for demo)
			return hex.EncodeToString(script[i : i+33])
		}
	}
	return ""
}
