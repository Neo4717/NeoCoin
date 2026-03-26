package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

func SHA256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func MerkleRoot(leaves [][]byte) ([]byte, error) {
	if len(leaves) == 0 {
		return nil, nil
	}
	level := make([][]byte, 0, len(leaves))
	for _, l := range leaves {
		if len(l) != 32 {
			return nil, nil
		}
		level = append(level, hashLeaf(l))
	}
	for len(level) > 1 {
		next := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, hashNode(left, right))
		}
		level = next
	}
	return append([]byte(nil), level[0]...), nil
}

func hashLeaf(leaf []byte) []byte {
	var b [1 + 32]byte
	b[0] = 0x00
	copy(b[1:], leaf)
	sum := sha256.Sum256(b[:])
	return sum[:]
}

func hashNode(left, right []byte) []byte {
	var b [1 + 32 + 32]byte
	b[0] = 0x01
	copy(b[1:], left)
	copy(b[33:], right)
	sum := sha256.Sum256(b[:])
	return sum[:]
}

func MerkleProofFromLeaves(leaves [][]byte, index int) (branch [][]byte, siblingLeft []bool, root []byte, err error) {
	if len(leaves) == 0 {
		return nil, nil, nil, errors.New("empty leaves")
	}
	if index < 0 || index >= len(leaves) {
		return nil, nil, nil, errors.New("index out of range")
	}
	level := make([][]byte, 0, len(leaves))
	for _, l := range leaves {
		if len(l) != 32 {
			return nil, nil, nil, errors.New("leaf must be 32 bytes")
		}
		level = append(level, hashLeaf(l))
	}

	idx := index
	for len(level) > 1 {
		var sib []byte
		var sibIsLeft bool
		if idx%2 == 0 {
			sibIsLeft = false
			if idx+1 < len(level) {
				sib = level[idx+1]
			} else {
				sib = level[idx]
			}
		} else {
			sibIsLeft = true
			sib = level[idx-1]
		}
		branch = append(branch, append([]byte(nil), sib...))
		siblingLeft = append(siblingLeft, sibIsLeft)

		next := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, hashNode(left, right))
		}
		level = next
		idx = idx / 2
	}
	return branch, siblingLeft, append([]byte(nil), level[0]...), nil
}

type MerkleProofLite struct {
	BlockHash string
	TxIndex   int
	TxHash    string
	Proof     []string
}

func CreateMerkleProof(txs []*Transaction, target *Transaction) *MerkleProofData {
	hashes := make([][]byte, len(txs))
	for i, tx := range txs {
		txID, _ := TxIDHex(*tx)
		hashes[i] = Hash256([]byte(txID))
	}

	targetIdx := -1
	for i, tx := range txs {
		txID, _ := TxIDHex(*tx)
		targetID, _ := TxIDHex(*target)
		if txID == targetID {
			targetIdx = i
			break
		}
	}

	if targetIdx < 0 {
		return nil
	}

	proof := buildMerkleProof(hashes, targetIdx)
	txID, _ := TxIDHex(*target)

	return &MerkleProofData{
		BlockHash: "",
		TxIndex:   targetIdx,
		TxHash:    txID,
		Proof:     proof,
	}
}

func buildMerkleProof(hashes [][]byte, targetIdx int) []string {
	var proof []string
	level := hashes
	idx := targetIdx

	for len(level) > 1 {
		if idx%2 == 0 {
			if idx+1 < len(level) {
				proof = append(proof, hex.EncodeToString(level[idx+1]))
			}
		} else {
			proof = append(proof, hex.EncodeToString(level[idx-1]))
		}

		idx = idx / 2
		var nextLevel [][]byte
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				combined := append(level[i], level[i+1]...)
				nextLevel = append(nextLevel, Hash256(combined))
			} else {
				nextLevel = append(nextLevel, level[i])
			}
		}
		level = nextLevel
	}

	return proof
}

func VerifyMerkleProof(p *MerkleProofData, merkleRoot string) bool {
	if p == nil || len(p.Proof) == 0 {
		return false
	}

	hash := Hash256([]byte(p.TxHash))
	idx := p.TxIndex

	for _, step := range p.Proof {
		stepHash, _ := hex.DecodeString(step)
		if idx%2 == 0 {
			combined := append(hash, stepHash...)
			hash = Hash256(combined)
		} else {
			combined := append(stepHash, hash...)
			hash = Hash256(combined)
		}
		idx = idx / 2
	}

	computedRoot := hex.EncodeToString(hash)
	return computedRoot == merkleRoot
}
