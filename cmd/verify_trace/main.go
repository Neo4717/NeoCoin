package main

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
)

func main() {
	targetHash := "f6f5431d25bbf7b12e8add9af5e3475c44a0a5b8"
	candidates := []string{
		"039817b5e42bcac28d467faf3eb8d99caf5c496ee1d5a71e85f89bf71538d31fbb",
		"0246f372c74e94ce9cb3e8cc1de7f4dfd903116fb6535a71d3fafa01753452018b",
		"038148146228e752f559633580f0d661d0f412790304f0f2b88b028c4d968c363f",
		"03a3a284a520d04a0b9497711b73ea27ba8f59bd06a0884b1120834c2cf94695a5",
		"03b3551ef1078eee2d88d90adbfc616cdd110c3dc3474bdb9c012a61b4b222e35d",
		"0253f190353e719a74332b1037202e0477b73c663b9289fe0a65affc05b31cd919",
		"0313fa373d41ec3bea101c9f697a38d27836f0a42bacd7e09aa7d0d49dbe6dbd47",
		"0240398c32100a931e21bcb6a008f0001e640cd3e3e2077c5e627fa79c1b42528f",
		"03072004cd04b7ac923cf1ff1b3d00e07f37d85bf668a66157f2086d99f40230ef",
		"032ddd94d0ce39c480014473aff255888a2a494a60fde9fa61dcdf4ca20e5bc706",
		"0202f80a83274385f75c24e04e0b47d45bfdb0fd4614084e7e20cc779d20188cc1",
		"0357b5397f2cbc8ca8201e7e8d05c86904bca64068b1f03dba7943555840338d47",
		"03988cb33d7bb6ef28293c17a0f48e6210ac9a5a074c9e064175915f5018e3285d",
		"020b683fe45faef7631fb5e9a7b420f599487308ddfdbdce8bbec7e16cf018376a",
		"033055ad5087044775129386be7d3a463b5c2da11d01c6fc820519488d217950d5",
		"029111f75145c2d37a8bf7dd95a76d6adc3d07e9800a17824776fd6e9eb425f9f4",
		"026644e0988457268690dcec2a95ac3ff72bf7cd9391aaa31dbece0d5bd083afa5",
		"02ca9c988bc7930378124de82323313b63ac71105d0c1ad96aafafc436c802cfb8",
		"03d41318b012102416e5c98ef14620fdfa557b2b9ec278e9215c18bba2fdd6b954",
		"0324878cf71969d562c8d35fd0f02c5f2b4f9b9fc8f9e08b6643f18d8a6c8204e3",
		"03328eb92a45ae936e1c966ccdf48cb064dbeec238c49f7703aeef9c7bf8c8ea55",
		"02fba7a69f53850585d5bdcb4653e3570c1d461ee17672de2d17b82ce286a55d79",
	}

	fmt.Println("--- Candidate Verification for Puzzle 71 ---")
	fmt.Printf("Target Hash160: %s\n\n", targetHash)

	for _, pkHex := range candidates {
		pkBytes, err := hex.DecodeString(pkHex)
		if err != nil {
			continue
		}

		// Calculate Hash160: RIPEMD160(SHA256(PubKey))
		hash := btcutil.Hash160(pkBytes)
		calcHashHex := hex.EncodeToString(hash)

		if calcHashHex == targetHash {
			fmt.Printf("\n[🚨] SUCCESS! PUBKEY MATCH FOUND!\n")
			fmt.Printf("PubKey: %s\n", pkHex)
			fmt.Printf("You can now use Pollard's Kangaroo to crack the 7.1 BTC reward.\n")
			return
		} else {
			fmt.Printf("[-] No match: %s... (Result: %s...)\n", pkHex[:16], calcHashHex[:16])
		}
	}

	fmt.Println("\nVerification finished. No direct matches found in this candidate set.")
}
