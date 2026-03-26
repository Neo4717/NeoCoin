package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ConsensusParams struct {
	DifficultyEnable bool

	TargetBlockTime   time.Duration
	DifficultyWindow  int
	DifficultyMaxStep uint32

	MinDifficultyBits uint32
	MaxDifficultyBits uint32

	GenesisDifficultyBits uint32

	MedianTimePastWindow int
	MaxTimeDrift         int64
	MaxBlockSize         uint64

	MerkleEnable           bool
	MerkleActivationHeight uint64

	BinaryEncodingEnable           bool
	BinaryEncodingActivationHeight uint64

	MonetaryPolicy MonetaryPolicy
}

func defaultConsensusParamsFromEnv() ConsensusParams {
	p := ConsensusParams{
		DifficultyEnable:               envBool("DIFFICULTY_ENABLE", false),
		TargetBlockTime:                envDurationMS("DIFFICULTY_TARGET_MS", 15*time.Second),
		DifficultyWindow:               envInt("DIFFICULTY_WINDOW", 20),
		DifficultyMaxStep:              envUint32("DIFFICULTY_MAX_STEP", 1),
		MinDifficultyBits:              envUint32("DIFFICULTY_MIN_BITS", 1),
		MaxDifficultyBits:              envUint32("DIFFICULTY_MAX_BITS", 255),
		GenesisDifficultyBits:          envUint32("GENESIS_DIFFICULTY_BITS", defaultDifficultyBits),
		MedianTimePastWindow:           envInt("MTP_WINDOW", 11),
		MaxTimeDrift:                   envInt64("MAX_TIME_DRIFT", envInt64("MAX_FUTURE_DRIFT_SEC", 2*60*60)),
		MaxBlockSize:                   envUint64("MAX_BLOCK_SIZE", 1_000_000),
		MerkleEnable:                   envBool("MERKLE_ENABLE", false),
		MerkleActivationHeight:         envUint64("MERKLE_ACTIVATION_HEIGHT", 0),
		BinaryEncodingEnable:           envBool("BINARY_ENCODING_ENABLE", false),
		BinaryEncodingActivationHeight: envUint64("BINARY_ENCODING_ACTIVATION_HEIGHT", 0),
	}

	if p.TargetBlockTime <= 0 {
		p.TargetBlockTime = 15 * time.Second
	}
	if p.DifficultyWindow <= 0 {
		p.DifficultyWindow = 20
	}
	if p.DifficultyMaxStep == 0 {
		p.DifficultyMaxStep = 1
	}
	if p.MinDifficultyBits == 0 {
		p.MinDifficultyBits = 1
	}
	if p.MaxDifficultyBits == 0 {
		p.MaxDifficultyBits = 255
	}
	if p.MaxDifficultyBits > maxDifficultyBits {
		p.MaxDifficultyBits = maxDifficultyBits
	}
	if p.MinDifficultyBits > p.MaxDifficultyBits {
		p.MinDifficultyBits = p.MaxDifficultyBits
	}
	if p.GenesisDifficultyBits < p.MinDifficultyBits {
		p.GenesisDifficultyBits = p.MinDifficultyBits
	}
	if p.GenesisDifficultyBits > p.MaxDifficultyBits {
		p.GenesisDifficultyBits = p.MaxDifficultyBits
	}
	if p.MedianTimePastWindow <= 0 {
		p.MedianTimePastWindow = 11
	}
	if p.MaxTimeDrift <= 0 {
		p.MaxTimeDrift = 2 * 60 * 60
	}
	if p.MaxBlockSize == 0 {
		p.MaxBlockSize = 1_000_000
	}
	return p
}

func (p ConsensusParams) BinaryEncodingActive(height uint64) bool {
	return p.BinaryEncodingEnable && height >= p.BinaryEncodingActivationHeight
}

func (bc *Blockchain) NextDifficultyBits() uint32 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return nextDifficultyBitsFromPath(bc.consensus, bc.blocks)
}

const maxDifficultyBits = uint32(256)

const defaultDifficultyBits = uint32(18)

func nextDifficultyBitsFromPath(p ConsensusParams, path []*Block) uint32 {
	if len(path) == 0 {
		return p.GenesisDifficultyBits
	}
	parentIdx := len(path) - 1
	parent := path[parentIdx]

	if !p.DifficultyEnable {
		if parent.DifficultyBits == 0 {
			return p.GenesisDifficultyBits
		}
		return clampDifficultyBits(p, parent.DifficultyBits)
	}

	if parentIdx < p.DifficultyWindow {
		return clampDifficultyBits(p, parent.DifficultyBits)
	}

	older := path[parentIdx-p.DifficultyWindow]
	actualSpanSec := parent.TimestampUnix - older.TimestampUnix
	if actualSpanSec <= 0 {
		actualSpanSec = 1
	}

	targetSec := int64(p.TargetBlockTime / time.Second)
	if targetSec <= 0 {
		targetSec = 1
	}
	expectedSpanSec := int64(p.DifficultyWindow) * targetSec
	if expectedSpanSec <= 0 {
		expectedSpanSec = 1
	}
	if actualSpanSec < expectedSpanSec/4 {
		actualSpanSec = expectedSpanSec / 4
	}
	if actualSpanSec > expectedSpanSec*4 {
		actualSpanSec = expectedSpanSec * 4
	}

	next := int64(parent.DifficultyBits)
	if actualSpanSec < expectedSpanSec/2 {
		next += int64(p.DifficultyMaxStep)
	} else if actualSpanSec > expectedSpanSec*2 {
		next -= int64(p.DifficultyMaxStep)
	}

	if next < 1 {
		next = 1
	}
	return clampDifficultyBits(p, uint32(next))
}

func expectedDifficultyBitsForBlockIndex(p ConsensusParams, path []*Block, idx int) uint32 {
	if idx <= 0 || idx >= len(path) {
		return 0
	}
	parentPath := path[:idx]
	return nextDifficultyBitsFromPath(p, parentPath)
}

func clampDifficultyBits(p ConsensusParams, bits uint32) uint32 {
	if bits < 1 {
		bits = 1
	}
	if bits > maxDifficultyBits {
		bits = maxDifficultyBits
	}
	if bits < p.MinDifficultyBits {
		return p.MinDifficultyBits
	}
	if bits > p.MaxDifficultyBits {
		return p.MaxDifficultyBits
	}
	return bits
}

func validateBlockTime(p ConsensusParams, path []*Block, idx int) error {
	if idx <= 0 || idx >= len(path) {
		return nil
	}
	prev := path[idx-1]
	cur := path[idx]

	if cur.TimestampUnix <= prev.TimestampUnix {
		return fmt.Errorf("timestamp not increasing at height %d", cur.Height)
	}

	mtp := medianTimePast(p, path, idx-1)
	if cur.TimestampUnix <= mtp {
		return fmt.Errorf("timestamp too old at height %d", cur.Height)
	}

	if p.MaxTimeDrift > 0 && cur.TimestampUnix > time.Now().Unix()+p.MaxTimeDrift {
		return fmt.Errorf("timestamp too far in future at height %d", cur.Height)
	}
	return nil
}

func medianTimePast(p ConsensusParams, path []*Block, endIdx int) int64 {
	window := p.MedianTimePastWindow
	if window <= 0 {
		window = 11
	}
	if endIdx < 0 {
		return 0
	}
	start := endIdx - (window - 1)
	if start < 0 {
		start = 0
	}
	ts := make([]int64, 0, endIdx-start+1)
	for i := start; i <= endIdx && i < len(path); i++ {
		ts = append(ts, path[i].TimestampUnix)
	}
	if len(ts) == 0 {
		return 0
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
	return ts[len(ts)/2]
}

func blockVersionForHeight(p ConsensusParams, height uint64) uint32 {
	if p.MerkleEnable && height >= p.MerkleActivationHeight {
		return 2
	}
	return 1
}

func blockSizeForConsensus(b *Block) (int, error) {
	if b == nil {
		return 0, errors.New("nil block")
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return 0, err
	}
	return len(raw), nil
}

func envBool(name string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	v = strings.ToLower(v)
	return v == "1" || v == "true" || v == "yes" || v == "y"
}

func envInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envUint32(name string, def uint32) uint32 {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return def
	}
	return uint32(n)
}

func envUint64(name string, def uint64) uint64 {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func envInt64(name string, def int64) int64 {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func envDurationMS(name string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	ms, err := strconv.Atoi(v)
	if err != nil || ms <= 0 {
		return def
	}
	return time.Duration(ms) * time.Millisecond
}

func consensusEnvOverridesSet() bool {
	names := []string{
		"DIFFICULTY_ENABLE",
		"DIFFICULTY_TARGET_MS",
		"DIFFICULTY_WINDOW",
		"DIFFICULTY_MAX_STEP",
		"DIFFICULTY_MIN_BITS",
		"DIFFICULTY_MAX_BITS",
		"GENESIS_DIFFICULTY_BITS",
		"MTP_WINDOW",
		"MAX_TIME_DRIFT",
		"MAX_FUTURE_DRIFT_SEC",
		"MAX_BLOCK_SIZE",
		"MERKLE_ENABLE",
		"MERKLE_ACTIVATION_HEIGHT",
		"BINARY_ENCODING_ENABLE",
		"BINARY_ENCODING_ACTIVATION_HEIGHT",
	}
	for _, name := range names {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}

const rulesHashVersionV3 = uint8(3)

func (p ConsensusParams) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := buf.WriteByte(rulesHashVersionV3); err != nil {
		return nil, err
	}
	if err := writeBool(&buf, p.DifficultyEnable); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, int64(p.TargetBlockTime)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(p.DifficultyWindow)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.DifficultyMaxStep); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MinDifficultyBits); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MaxDifficultyBits); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.GenesisDifficultyBits); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(p.MedianTimePastWindow)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MaxTimeDrift); err != nil {
		return nil, err
	}
	if err := writeBool(&buf, p.MerkleEnable); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MerkleActivationHeight); err != nil {
		return nil, err
	}
	if err := writeBool(&buf, p.BinaryEncodingEnable); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.BinaryEncodingActivationHeight); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MaxBlockSize); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MonetaryPolicy.InitialBlockReward); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MonetaryPolicy.HalvingInterval); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(p.MonetaryPolicy.MinerFeeShare); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, p.MonetaryPolicy.TailEmission); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p ConsensusParams) RulesHash() ([32]byte, error) {
	preimage, err := p.MarshalBinary()
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(preimage), nil
}

func (p ConsensusParams) MustRulesHash() [32]byte {
	h, err := p.RulesHash()
	if err != nil {
		panic(fmt.Sprintf("failed to compute rules hash: %v", err))
	}
	return h
}

func writeBool(buf *bytes.Buffer, v bool) error {
	if v {
		return buf.WriteByte(1)
	}
	return buf.WriteByte(0)
}

type Uint64String uint64

func (u *Uint64String) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return errors.New("invalid uint64: empty")
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return errors.New("invalid uint64: empty string")
		}
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint64 string: %w", err)
		}
		*u = Uint64String(v)
		return nil
	}
	var n json.Number
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&n); err != nil {
		return err
	}
	v, err := strconv.ParseUint(n.String(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid uint64 number: %w", err)
	}
	*u = Uint64String(v)
	return nil
}

func (u Uint64String) Uint64() uint64 {
	return uint64(u)
}

type genesisConfigJSON struct {
	Network             string          `json:"network"`
	ChainID             uint64          `json:"chainId"`
	Timestamp           int64           `json:"timestamp"`
	GenesisMinerAddress string          `json:"genesisMinerAddress"`
	InitialSupply       Uint64String    `json:"initialSupply"`
	GenesisMessage      string          `json:"genesisMessage,omitempty"`
	MonetaryPolicy      json.RawMessage `json:"monetaryPolicy"`
	ConsensusParams     json.RawMessage `json:"consensusParams"`
}

type GenesisConfig struct {
	Network             string
	ChainID             uint64
	Timestamp           int64
	GenesisMinerAddress string
	InitialSupply       uint64
	GenesisMessage      string
	MonetaryPolicy      MonetaryPolicy
	ConsensusParams     ConsensusParams
}

func GenesisPathFromEnv(chainID uint64) (string, error) {
	if path := strings.TrimSpace(os.Getenv("GENESIS_PATH")); path != "" {
		return path, nil
	}
	if network := strings.TrimSpace(os.Getenv("GENESIS_NETWORK")); network != "" {
		return filepath.Join("genesis", network+".json"), nil
	}
	if network := strings.TrimSpace(os.Getenv("NETWORK")); network != "" {
		return filepath.Join("genesis", network+".json"), nil
	}
	switch chainID {
	case 0, 1:
		return filepath.Join("genesis", "mainnet.json"), nil
	case 2:
		return filepath.Join("genesis", "testnet.json"), nil
	default:
		return "", fmt.Errorf("GENESIS_PATH is required for chainId=%d", chainID)
	}
}

func LoadGenesisConfig(path string) (*GenesisConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read genesis file %s: %w", path, err)
	}

	var raw genesisConfigJSON
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse genesis config: %w", err)
	}
	if raw.ChainID == 0 {
		return nil, errors.New("genesis chainId must be > 0")
	}
	if raw.Timestamp <= 0 {
		return nil, errors.New("genesis timestamp must be > 0")
	}
	if raw.InitialSupply.Uint64() == 0 {
		return nil, errors.New("genesis initialSupply must be > 0")
	}
	if err := validateGenesisMinerAddress(raw.GenesisMinerAddress); err != nil {
		return nil, fmt.Errorf("invalid genesisMinerAddress: %w", err)
	}

	policy, err := parseMonetaryPolicy(raw.MonetaryPolicy)
	if err != nil {
		return nil, err
	}
	consensus, err := parseConsensusParams(raw.ConsensusParams)
	if err != nil {
		return nil, err
	}
	consensus.MonetaryPolicy = policy

	cfg := &GenesisConfig{
		Network:             raw.Network,
		ChainID:             raw.ChainID,
		Timestamp:           raw.Timestamp,
		GenesisMinerAddress: raw.GenesisMinerAddress,
		InitialSupply:       raw.InitialSupply.Uint64(),
		GenesisMessage:      raw.GenesisMessage,
		MonetaryPolicy:      policy,
		ConsensusParams:     consensus,
	}
	return cfg, nil
}

func validateGenesisMinerAddress(addr string) error {
	if strings.HasPrefix(addr, "NEO") {
		return ValidateAddress(addr)
	}
	b, err := hex.DecodeString(addr)
	if err != nil {
		return fmt.Errorf("invalid hex: %w", err)
	}
	if len(b) != 32 {
		return fmt.Errorf("raw address must be 32 bytes, got %d", len(b))
	}
	return nil
}

func BuildGenesisBlock(cfg *GenesisConfig, consensus ConsensusParams) (*Block, error) {
	if cfg == nil {
		return nil, errors.New("missing genesis config")
	}
	msg := genesisMessageOrDefault(cfg)
	coinbase := Transaction{
		Type:      TxCoinbase,
		ChainID:   cfg.ChainID,
		ToAddress: cfg.GenesisMinerAddress,
		Amount:    cfg.InitialSupply,
		Data:      msg,
	}
	genesis := &Block{
		Version:        blockVersionForHeight(consensus, 0),
		Height:         0,
		TimestampUnix:  cfg.Timestamp,
		DifficultyBits: consensus.GenesisDifficultyBits,
		MinerAddress:   cfg.GenesisMinerAddress,
		Transactions:   []Transaction{coinbase},
	}
	pow := NewProofOfWork(consensus, genesis)
	nonce, hash, err := pow.Run()
	if err != nil {
		return nil, err
	}
	genesis.Nonce = nonce
	genesis.Hash = hash
	return genesis, nil
}

func ValidateGenesisBlock(b *Block, cfg *GenesisConfig, consensus ConsensusParams) error {
	if b == nil {
		return errors.New("missing genesis block")
	}
	if b.Height != 0 {
		return fmt.Errorf("invalid genesis height: %d", b.Height)
	}
	if len(b.PrevHash) != 0 {
		return errors.New("invalid genesis prevHash")
	}
	if b.Version != blockVersionForHeight(consensus, 0) {
		return fmt.Errorf("invalid genesis version: %d", b.Version)
	}
	if b.TimestampUnix != cfg.Timestamp {
		return fmt.Errorf("genesis timestamp mismatch: %d != %d", b.TimestampUnix, cfg.Timestamp)
	}
	if b.MinerAddress != cfg.GenesisMinerAddress {
		return fmt.Errorf("genesis miner mismatch: %s != %s", b.MinerAddress, cfg.GenesisMinerAddress)
	}
	if b.DifficultyBits != consensus.GenesisDifficultyBits {
		return fmt.Errorf("genesis difficulty mismatch: %d != %d", b.DifficultyBits, consensus.GenesisDifficultyBits)
	}
	if len(b.Transactions) != 1 {
		return errors.New("genesis must contain exactly one transaction")
	}
	cb := b.Transactions[0]
	if cb.Type != TxCoinbase {
		return errors.New("genesis tx must be coinbase")
	}
	if cb.ChainID != cfg.ChainID {
		return fmt.Errorf("genesis coinbase chainId mismatch: %d != %d", cb.ChainID, cfg.ChainID)
	}
	if cb.ToAddress != cfg.GenesisMinerAddress {
		return fmt.Errorf("genesis coinbase toAddress mismatch: %s != %s", cb.ToAddress, cfg.GenesisMinerAddress)
	}
	if cb.Amount != cfg.InitialSupply {
		return fmt.Errorf("genesis supply mismatch: %d != %d", cb.Amount, cfg.InitialSupply)
	}
	if cb.Data != genesisMessageOrDefault(cfg) {
		return fmt.Errorf("genesis message mismatch: %q != %q", cb.Data, genesisMessageOrDefault(cfg))
	}
	ok, err := NewProofOfWork(consensus, b).Validate()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("invalid genesis pow")
	}
	_, err = ensureBlockHash(b, consensus)
	return err
}

func ensureBlockHash(b *Block, consensus ConsensusParams) ([]byte, error) {
	header, err := b.HeaderBytesForConsensus(consensus, b.Nonce)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(header)
	if len(b.Hash) == 0 {
		b.Hash = append([]byte(nil), sum[:]...)
	}
	return sum[:], nil
}

func genesisMessageOrDefault(cfg *GenesisConfig) string {
	if cfg.GenesisMessage != "" {
		return cfg.GenesisMessage
	}
	return fmt.Sprintf("genesis allocation (supply=%d)", cfg.InitialSupply)
}
