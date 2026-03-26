package blockchain

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	dataDirName       = "data"
	blocksGobRel      = "data/blocks.gob"
	chainBoltRel      = "data/chain.db"
	rulesHashRel      = "data/rules.hash"
	genesisHashRel    = "data/genesis.hash"
	blocksBucket      = "blocks"
	canonBucket       = "canonical"
	metaBucket        = "meta"
	checkpointsBucket = "checkpoints"
	commitmentsBucket = "commitments"
	metaTipHash       = "tipHash"
	metaTipHeight     = "tipHeight"
	metaRulesHash     = "rulesHash"
	metaGenesisHash   = "genesisHash"
)

type ChainStore interface {
	ReadCanonical() ([]*Block, error)
	AppendCanonical(block *Block) error
	RewriteCanonical(blocks []*Block) error
	PutBlock(block *Block) error
	ReadAllBlocks() (map[string]*Block, error)
	GetRulesHash() ([]byte, bool, error)
	PutRulesHash(hash []byte) error
	GetGenesisHash() ([]byte, bool, error)
	PutGenesisHash(hash []byte) error
	PruneBelow(height int64) error
	WriteCheckpoint(height int64, state map[string]Account) error
	ReadCheckpoint(height int64) (map[string]Account, error)
	GetCheckpointHeights() ([]int64, error)
	WriteCommitment(c *StateCommitment) error
	ReadCommitment(height int64) (*StateCommitment, error)
}

func OpenChainStoreFromEnv() (ChainStore, error) {
	backend := os.Getenv("STORE_BACKEND")
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}
	switch backend {
	case "", "bolt":
		boltPath := filepath.Join(dataDir, chainBoltRel)
		return OpenBoltChainStore(boltPath)
	default:
		return nil, fmt.Errorf("unknown STORE_BACKEND: %q", backend)
	}
}

var _ ChainStore = (*BoltChainStore)(nil)

type BoltChainStore struct {
	DB   *bolt.DB
	path string
}

func OpenBoltChainStore(path string) (*BoltChainStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	s := &BoltChainStore{DB: db, path: path}
	if err := s.DB.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(blocksBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(canonBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(metaBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(checkpointsBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(commitmentsBucket)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *BoltChainStore) Close() error {
	return s.DB.Close()
}

func (s *BoltChainStore) PutBlock(block *Block) error {
	if block == nil || len(block.Hash) == 0 {
		return errors.New("missing block hash")
	}
	key := append([]byte(nil), block.Hash...)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(block); err != nil {
		return err
	}
	val := buf.Bytes()

	return s.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if existing := b.Get(key); existing != nil {
			return nil
		}
		return b.Put(key, val)
	})
}

func (s *BoltChainStore) ReadAllBlocks() (map[string]*Block, error) {
	out := map[string]*Block{}
	err := s.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		return b.ForEach(func(k, v []byte) error {
			var blk Block
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&blk); err != nil {
				return err
			}
			h := hex.EncodeToString(k)
			bc := blk
			out[h] = &bc
			return nil
		})
	})
	return out, err
}

func (s *BoltChainStore) ReadCanonical() ([]*Block, error) {
	var blocks []*Block
	err := s.DB.View(func(tx *bolt.Tx) error {
		canonB := tx.Bucket([]byte(canonBucket))
		c := canonB.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			hash := v
			raw := tx.Bucket([]byte(blocksBucket)).Get(hash)
			if raw == nil {
				return errors.New("canonical block missing")
			}
			var blk Block
			if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&blk); err != nil {
				return err
			}
			bc := blk
			blocks = append(blocks, &bc)
		}
		return nil
	})
	return blocks, err
}

func (s *BoltChainStore) AppendCanonical(block *Block) error {
	if block == nil || len(block.Hash) == 0 {
		return errors.New("missing block hash")
	}
	heightKey := u64beBigEndian(block.Height)
	hashKey := append([]byte(nil), block.Hash...)

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(block); err != nil {
		return err
	}
	val := buf.Bytes()

	err := s.DB.Update(func(tx *bolt.Tx) error {
		blocksB := tx.Bucket([]byte(blocksBucket))
		canonB := tx.Bucket([]byte(canonBucket))
		metaB := tx.Bucket([]byte(metaBucket))

		if existing := blocksB.Get(hashKey); existing == nil {
			if err := blocksB.Put(hashKey, val); err != nil {
				return err
			}
		}

		if block.Height > 0 {
			prevHash := canonB.Get(u64beBigEndian(block.Height - 1))
			if prevHash == nil {
				return errors.New("missing previous canonical height")
			}
			if !bytes.Equal(prevHash, block.PrevHash) {
				return errors.New("prevhash mismatch for append")
			}
		}

		if err := canonB.Put(heightKey, hashKey); err != nil {
			return err
		}
		if err := metaB.Put([]byte(metaTipHash), hashKey); err != nil {
			return err
		}
		return metaB.Put([]byte(metaTipHeight), heightKey)
	})
	return err
}

func (s *BoltChainStore) RewriteCanonical(blocks []*Block) error {
	return s.DB.Update(func(tx *bolt.Tx) error {
		blocksB := tx.Bucket([]byte(blocksBucket))
		canonB := tx.Bucket([]byte(canonBucket))
		metaB := tx.Bucket([]byte(metaBucket))

		c := canonB.Cursor()
		for k, _ := c.First(); k != nil; {
			nextK, _ := c.Next()
			if err := canonB.Delete(k); err != nil {
				return err
			}
			k = nextK
		}

		var tipHash []byte
		var tipHeight uint64
		for _, b := range blocks {
			if b == nil || len(b.Hash) == 0 {
				return errors.New("missing block hash")
			}
			var buf bytes.Buffer
			if err := gob.NewEncoder(&buf).Encode(b); err != nil {
				return err
			}
			key := append([]byte(nil), b.Hash...)
			if existing := blocksB.Get(key); existing == nil {
				if err := blocksB.Put(key, buf.Bytes()); err != nil {
					return err
				}
			}
			if err := canonB.Put(u64beBigEndian(b.Height), key); err != nil {
				return err
			}
			tipHash = key
			tipHeight = b.Height
		}
		if tipHash == nil {
			_ = metaB.Delete([]byte(metaTipHash))
			_ = metaB.Delete([]byte(metaTipHeight))
			return nil
		}
		if err := metaB.Put([]byte(metaTipHash), tipHash); err != nil {
			return err
		}
		return metaB.Put([]byte(metaTipHeight), u64beBigEndian(tipHeight))
	})
}

var _ encoding.BinaryMarshaler = (*bigEndian)(nil)

type bigEndian struct{}

func (bigEndian) MarshalBinary() ([]byte, error) {
	return []byte{0}, nil
}

func u64beBigEndian(v uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	return b[:]
}

func (s *BoltChainStore) GetRulesHash() ([]byte, bool, error) {
	var out []byte
	err := s.DB.View(func(tx *bolt.Tx) error {
		metaB := tx.Bucket([]byte(metaBucket))
		if metaB == nil {
			return nil
		}
		v := metaB.Get([]byte(metaRulesHash))
		if v == nil {
			return nil
		}
		out = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func (s *BoltChainStore) PutRulesHash(hash []byte) error {
	if len(hash) != 32 {
		return errors.New("rules hash must be 32 bytes")
	}
	return s.DB.Update(func(tx *bolt.Tx) error {
		metaB := tx.Bucket([]byte(metaBucket))
		if metaB == nil {
			var err error
			metaB, err = tx.CreateBucketIfNotExists([]byte(metaBucket))
			if err != nil {
				return err
			}
		}
		return metaB.Put([]byte(metaRulesHash), append([]byte(nil), hash...))
	})
}

func (s *BoltChainStore) GetGenesisHash() ([]byte, bool, error) {
	var out []byte
	err := s.DB.View(func(tx *bolt.Tx) error {
		metaB := tx.Bucket([]byte(metaBucket))
		if metaB == nil {
			return nil
		}
		v := metaB.Get([]byte(metaGenesisHash))
		if v == nil {
			return nil
		}
		out = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func (s *BoltChainStore) PutGenesisHash(hash []byte) error {
	if len(hash) != 32 {
		return errors.New("genesis hash must be 32 bytes")
	}
	return s.DB.Update(func(tx *bolt.Tx) error {
		metaB := tx.Bucket([]byte(metaBucket))
		if metaB == nil {
			var err error
			metaB, err = tx.CreateBucketIfNotExists([]byte(metaBucket))
			if err != nil {
				return err
			}
		}
		return metaB.Put([]byte(metaGenesisHash), append([]byte(nil), hash...))
	})
}

func (s *BoltChainStore) PruneBelow(height int64) error {
	return s.DB.Update(func(tx *bolt.Tx) error {
		canonB := tx.Bucket([]byte(canonBucket))
		blocksB := tx.Bucket([]byte(blocksBucket))

		type entry struct {
			heightKey []byte
			hashKey   []byte
		}
		var toDelete []entry

		c := canonB.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if len(k) != 8 {
				continue
			}
			h := int64(binary.BigEndian.Uint64(k))
			if h > height {
				break
			}
			if h == height {
				continue
			}
			toDelete = append(toDelete, entry{heightKey: append([]byte(nil), k...), hashKey: append([]byte(nil), v...)})
		}

		for _, e := range toDelete {
			if err := canonB.Delete(e.heightKey); err != nil {
				return err
			}
			_ = blocksB.Delete(e.hashKey)
		}

		return nil
	})
}

func (s *BoltChainStore) WriteCheckpoint(height int64, state map[string]Account) error {
	key := u64beBigEndian(uint64(height))

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return s.DB.Update(func(tx *bolt.Tx) error {
		checkpointsB := tx.Bucket([]byte(checkpointsBucket))
		return checkpointsB.Put(key, data)
	})
}

func (s *BoltChainStore) ReadCheckpoint(height int64) (map[string]Account, error) {
	key := u64beBigEndian(uint64(height))
	var state map[string]Account

	err := s.DB.View(func(tx *bolt.Tx) error {
		checkpointsB := tx.Bucket([]byte(checkpointsBucket))
		v := checkpointsB.Get(key)
		if v == nil {
			return fmt.Errorf("checkpoint not found at height %d", height)
		}
		if err := json.Unmarshal(v, &state); err != nil {
			return err
		}
		return nil
	})

	return state, err
}

func (s *BoltChainStore) GetCheckpointHeights() ([]int64, error) {
	var heights []int64

	err := s.DB.View(func(tx *bolt.Tx) error {
		checkpointsB := tx.Bucket([]byte(checkpointsBucket))
		c := checkpointsB.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if len(k) == 8 {
				heights = append(heights, int64(binary.BigEndian.Uint64(k)))
			}
		}
		return nil
	})

	sortInts(heights)
	return heights, err
}

func sortInts(a []int64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

func (s *BoltChainStore) WriteCommitment(c *StateCommitment) error {
	return s.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(commitmentsBucket))
		if bucket == nil {
			var err error
			bucket, err = tx.CreateBucketIfNotExists([]byte(commitmentsBucket))
			if err != nil {
				return err
			}
		}

		data, err := json.Marshal(c)
		if err != nil {
			return err
		}

		var heightBytes [8]byte
		binary.BigEndian.PutUint64(heightBytes[:], uint64(c.BlockHeight))

		return bucket.Put(heightBytes[:], data)
	})
}

func (s *BoltChainStore) ReadCommitment(height int64) (*StateCommitment, error) {
	var commitment *StateCommitment

	err := s.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(commitmentsBucket))
		if bucket == nil {
			return fmt.Errorf("no commitments bucket")
		}

		var heightBytes [8]byte
		binary.BigEndian.PutUint64(heightBytes[:], uint64(height))

		data := bucket.Get(heightBytes[:])
		if data == nil {
			return fmt.Errorf("commitment not found at height %d", height)
		}

		return json.Unmarshal(data, &commitment)
	})

	return commitment, err
}
