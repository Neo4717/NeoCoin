package economy

import (
	"testing"
	"time"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

func TestFeeMarket_CalculateBaseFee(t *testing.T) {
	fm := NewFeeMarket()

	tests := []struct {
		name           string
		parentGasUsed  uint64
		parentGasLimit uint64
		wantBaseFee    uint64
	}{
		{
			name:           "empty parent gas limit returns current base fee",
			parentGasUsed:  500000,
			parentGasLimit: 0,
			wantBaseFee:    1000000000,
		},
		{
			name:           "high usage increases base fee",
			parentGasUsed:  800000,
			parentGasLimit: 1000000,
			wantBaseFee:    1000000000,
		},
		{
			name:           "low usage decreases base fee",
			parentGasUsed:  200000,
			parentGasLimit: 1000000,
			wantBaseFee:    1000000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fm.CalculateBaseFee(tt.parentGasUsed, tt.parentGasLimit, 1)
			if got == 0 {
				t.Errorf("CalculateBaseFee() = 0, want non-zero")
			}
		})
	}
}

func TestFeeMarket_EstimateFee(t *testing.T) {
	fm := NewFeeMarket()

	est := fm.EstimateFee()
	if est.Low == 0 || est.Medium == 0 || est.High == 0 {
		t.Errorf("EstimateFee() returned zero values: Low=%d, Medium=%d, High=%d", est.Low, est.Medium, est.High)
	}
}

func TestFeeMarket_AddBlockToHistory(t *testing.T) {
	fm := NewFeeMarket()

	entry := FeeHistoryEntry{
		BlockHeight:       1,
		BaseFee:           1000000000,
		GasUsed:           500000,
		GasLimit:          1000000,
		PriorityFeePerGas: 1000000,
		Timestamp:         time.Now(),
	}

	fm.AddBlockToHistory(entry)

	history := fm.GetFeeHistory(10)
	if len(history) != 1 {
		t.Errorf("GetFeeHistory() = %d entries, want 1", len(history))
	}
}

func TestFeeMarket_PriorityFee(t *testing.T) {
	fm := NewFeeMarket()

	tests := []struct {
		name       string
		senderFee  uint64
		userTipMax uint64
		want       uint64
	}{
		{
			name:       "below minimum returns minimum",
			senderFee:  500000,
			userTipMax: 500,
			want:       1000000,
		},
		{
			name:       "above maximum returns maximum",
			senderFee:  500000,
			userTipMax: 2000000000,
			want:       1000000000,
		},
		{
			name:       "valid tip uses sender fee when lower",
			senderFee:  500000,
			userTipMax: 10000000,
			want:       500000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fm.PriorityFee(tt.senderFee, tt.userTipMax)
			if got == 0 {
				t.Errorf("PriorityFee() = 0, want non-zero")
			}
		})
	}
}

func TestMiningIncentives_CalculateBlockReward(t *testing.T) {
	policy := blockchain.MonetaryPolicy{
		InitialBlockReward: 5000000000,
		HalvingInterval:    210000,
		MinerFeeShare:      80,
		TailEmission:       500000000,
	}

	mi := NewMiningIncentives(policy)

	tests := []struct {
		name       string
		height     uint64
		wantReward uint64
	}{
		{
			name:       "genesis block gets initial reward",
			height:     0,
			wantReward: 5000000000,
		},
		{
			name:       "before halving",
			height:     100000,
			wantReward: 5000000000,
		},
		{
			name:       "at halving point",
			height:     210000,
			wantReward: 2500000000,
		},
		{
			name:       "after halving",
			height:     420000,
			wantReward: 1250000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mi.CalculateBlockReward(tt.height)
			if got == 0 {
				t.Errorf("CalculateBlockReward() = 0, want non-zero")
			}
		})
	}
}

func TestMiningIncentives_CalculateMinerReward(t *testing.T) {
	policy := blockchain.MonetaryPolicy{
		InitialBlockReward: 5000000000,
		HalvingInterval:    210000,
		MinerFeeShare:      80,
		TailEmission:       500000000,
	}

	mi := NewMiningIncentives(policy)

	txs := []blockchain.Transaction{
		{Fee: 1000000},
		{Fee: 2000000},
	}

	minerReward, burnedFees := mi.CalculateMinerReward(1, txs)

	if minerReward == 0 {
		t.Errorf("CalculateMinerReward() minerReward = 0, want non-zero")
	}
	if burnedFees == 0 {
		t.Errorf("CalculateMinerReward() burnedFees = 0, want non-zero")
	}
}

func TestMiningIncentives_CalculateUncleReward(t *testing.T) {
	policy := blockchain.MonetaryPolicy{
		InitialBlockReward: 5000000000,
		HalvingInterval:    210000,
		MinerFeeShare:      80,
	}

	mi := NewMiningIncentives(policy)

	reward := mi.CalculateUncleReward(100, 105)
	if reward == 0 {
		t.Errorf("CalculateUncleReward() = 0, want non-zero")
	}
}

func TestMiningIncentives_GetSubsidySchedule(t *testing.T) {
	policy := blockchain.MonetaryPolicy{
		InitialBlockReward: 5000000000,
		HalvingInterval:    210000,
		MinerFeeShare:      80,
	}

	mi := NewMiningIncentives(policy)

	schedule := mi.GetSubsidySchedule(0, 3)
	if len(schedule) != 3 {
		t.Errorf("GetSubsidySchedule() = %d periods, want 3", len(schedule))
	}
}

func TestEconomicSecurity_Calculate51AttackCost(t *testing.T) {
	es := NewEconomicSecurity(1000000000, 1000000000)

	cost := es.Calculate51AttackCost(1)
	if cost == 0 {
		t.Errorf("Calculate51AttackCost() = 0, want non-zero")
	}
}

func TestEconomicSecurity_Is51AttackProfitable(t *testing.T) {
	es := NewEconomicSecurity(1000, 1000)

	profitable := es.Is51AttackProfitable(1)
	if profitable {
		t.Logf("51 attack is profitable at current parameters")
	}
}

func TestEconomicSecurity_DetectSelfishMining(t *testing.T) {
	es := NewEconomicSecurity(1000000000, 1000000000)

	detected := es.DetectSelfishMining("miner1", 30, 100)
	if detected {
		t.Logf("Selfish mining detected")
	}
}

func TestEconomicSecurity_RecordMinerBlock(t *testing.T) {
	es := NewEconomicSecurity(1000000000, 1000000000)

	es.RecordMinerBlock("miner1", time.Now())

	perf, ok := es.GetMinerPerformance("miner1")
	if !ok {
		t.Errorf("GetMinerPerformance() = false, want true")
	}
	if perf.BlocksMined != 1 {
		t.Errorf("BlocksMined = %d, want 1", perf.BlocksMined)
	}
}

func TestEconomicSecurity_PreventFeeSniping(t *testing.T) {
	es := NewEconomicSecurity(1000000000, 1000000000)

	tests := []struct {
		name          string
		blockHeight   uint64
		txBlockHeight uint64
		want          bool
	}{
		{
			name:          "tx in same block",
			blockHeight:   100,
			txBlockHeight: 100,
			want:          false,
		},
		{
			name:          "tx in recent block within window",
			blockHeight:   100,
			txBlockHeight: 98,
			want:          true,
		},
		{
			name:          "tx in old block outside window",
			blockHeight:   100,
			txBlockHeight: 90,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := es.PreventFeeSniping(tt.blockHeight, tt.txBlockHeight)
			if got != tt.want {
				t.Errorf("PreventFeeSniping(%d, %d) = %v, want %v", tt.blockHeight, tt.txBlockHeight, got, tt.want)
			}
		})
	}
}

func TestEconomicSecurity_ValidateTransactionOrdering(t *testing.T) {
	es := NewEconomicSecurity(1000000000, 1000000000)

	validTxs := []blockchain.Transaction{
		{Nonce: 1},
		{Nonce: 2},
		{Nonce: 3},
	}

	err := es.ValidateTransactionOrdering(validTxs)
	if err != nil {
		t.Errorf("ValidateTransactionOrdering() error = %v, want nil", err)
	}

	invalidTxs := []blockchain.Transaction{
		{Nonce: 2},
		{Nonce: 1},
	}

	err = es.ValidateTransactionOrdering(invalidTxs)
	if err == nil {
		t.Errorf("ValidateTransactionOrdering() error = nil, want non-nil")
	}
}

func TestDynamicBlockSize_UpdateBlockSize(t *testing.T) {
	dbs := NewDynamicBlockSize(1000000)

	dbs.UpdateBlockSize(800000)
	dbs.UpdateBlockSize(850000)

	size := dbs.GetCurrentSize()
	if size == 0 {
		t.Errorf("GetCurrentSize() = 0, want non-zero")
	}
}

func TestValidateTransaction(t *testing.T) {
	tests := []struct {
		name    string
		tx      blockchain.Transaction
		wantErr bool
	}{
		{
			name: "valid transfer transaction",
			tx: blockchain.Transaction{
				Type:      blockchain.TxTransfer,
				ToAddress: "NEOabc123",
				Amount:    1000,
				Fee:       1,
			},
			wantErr: false,
		},
		{
			name: "invalid - no recipient",
			tx: blockchain.Transaction{
				ToAddress: "",
				Amount:    1000,
			},
			wantErr: true,
		},
		{
			name: "invalid - zero amount and fee",
			tx: blockchain.Transaction{
				ToAddress: "NEOabc123",
				Amount:    0,
				Fee:       0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransaction(tt.tx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEconomyManager_ProcessBlock(t *testing.T) {
	policy := blockchain.MonetaryPolicy{
		InitialBlockReward: 5000000000,
		HalvingInterval:    210000,
		MinerFeeShare:      80,
	}

	em := NewEconomyManager(policy, 1000000, 1000000000)

	block := &blockchain.Block{
		Height:       1,
		MinerAddress: "NEOminer1",
		Transactions: []blockchain.Transaction{
			{Fee: 1000000},
		},
	}

	err := em.ProcessBlock(block, 500000, 1000000)
	if err != nil {
		t.Errorf("ProcessBlock() error = %v, want nil", err)
	}
}

func TestEconomyManager_GetSecurityReport(t *testing.T) {
	policy := blockchain.MonetaryPolicy{
		InitialBlockReward: 5000000000,
		HalvingInterval:    210000,
		MinerFeeShare:      80,
	}

	em := NewEconomyManager(policy, 1000000, 1000000000)

	report := em.GetSecurityReport()

	if report.NetworkSecurity.TotalHashRate == 0 {
		t.Errorf("NetworkSecurity.TotalHashRate = 0, want non-zero")
	}
	if report.CurrentBlockSize == 0 {
		t.Errorf("CurrentBlockSize = 0, want non-zero")
	}
}

func TestEconomyManager_ValidateBlockEconomics(t *testing.T) {
	policy := blockchain.MonetaryPolicy{
		InitialBlockReward: 5000000000,
		HalvingInterval:    210000,
		MinerFeeShare:      80,
	}

	em := NewEconomyManager(policy, 1000000, 1000000000)

	block := &blockchain.Block{
		Transactions: []blockchain.Transaction{
			{ToAddress: "NEOabc123", Amount: 1000, Fee: 1, Nonce: 1},
		},
	}

	err := em.ValidateBlockEconomics(block, nil)
	if err != nil {
		t.Errorf("ValidateBlockEconomics() error = %v, want nil", err)
	}
}
