package mining

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type StratumServer struct {
	addr     string
	miner    *Miner
	listener net.Listener

	workers      sync.Map
	workerMu     sync.RWMutex
	nextWorkerID uint64

	jobs       sync.Map
	currentJob atomic.Value

	validShares    atomic.Int64
	invalidShares  atomic.Int64
	acceptedShares atomic.Int64

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type StratumWorker struct {
	ID       uint64
	conn     net.Conn
	addr     string
	user     string
	password string

	diff        atomic.Uint64
	diffHistory []uint64

	lastShare time.Time
	shares    atomic.Int64

	notifyMu   sync.Mutex
	subscribed bool
}

type StratumMessage struct {
	ID     interface{} `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

type MiningNotify struct {
	JobID        string   `json:"job_id"`
	PrevHash     string   `json:"prevhash"`
	Coinbase1    string   `json:"coinb1"`
	Coinbase2    string   `json:"coinb2"`
	MerkleBranch []string `json:"merkle_branch"`
	Version      string   `json:"version"`
	NBits        string   `json:"nbits"`
	NTime        string   `json:"ntime"`
	CleanJobs    bool     `json:"clean_jobs"`
}

type SubmitRequest struct {
	WorkerName  string `json:"worker_name"`
	JobID       string `json:"job_id"`
	ExtraNonce2 string `json:"extra_nonce2"`
	NTime       string `json:"ntime"`
	Nonce       string `json:"nonce"`
}

type SubscribeRequest struct {
	MinerAgent string `json:"miner_agent"`
}

func NewStratumServer(addr string, miner *Miner) *StratumServer {
	return &StratumServer{
		addr:   addr,
		miner:  miner,
		stopCh: make(chan struct{}),
	}
}

func (s *StratumServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = ln

	log.Printf("Stratum server listening on %s", s.addr)

	s.wg.Add(1)
	go s.acceptLoop(ctx)

	return nil
}

func (s *StratumServer) acceptLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			default:
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(ctx, conn)
	}
}

func (s *StratumServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	worker := &StratumWorker{
		ID:        atomic.AddUint64(&s.nextWorkerID, 1),
		conn:      conn,
		addr:      conn.RemoteAddr().String(),
		diff:      atomic.Uint64{},
		lastShare: time.Now(),
	}
	worker.diff.Store(1)

	s.workers.Store(worker.ID, worker)
	defer s.workers.Delete(worker.ID)

	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	dec := json.NewDecoder(conn)
	for {
		var msg StratumMessage
		if err := dec.Decode(&msg); err != nil {
			return
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		if err := s.handleMessage(worker, &msg); err != nil {
			log.Printf("worker %d error: %v", worker.ID, err)
			s.sendError(conn, msg.ID, -1, err.Error())
		}
	}
}

func (s *StratumServer) handleMessage(worker *StratumWorker, msg *StratumMessage) error {
	switch msg.Method {
	case "mining.subscribe":
		return s.handleSubscribe(worker, msg)
	case "mining.authorize":
		return s.handleAuthorize(worker, msg)
	case "mining.submit":
		return s.handleSubmit(worker, msg)
	case "mining.get_transactions":
		return s.handleGetTransactions(worker, msg)
	case "mining.extranonce.subscribe":
		return s.handleExtranonceSubscribe(worker, msg)
	default:
		return fmt.Errorf("unknown method: %s", msg.Method)
	}
}

func (s *StratumServer) handleSubscribe(worker *StratumWorker, msg *StratumMessage) error {
	var params json.RawMessage
	if p, ok := msg.Params.(json.RawMessage); ok {
		params = p
	} else if p, ok := msg.Params.([]byte); ok {
		params = p
	} else {
		return fmt.Errorf("invalid params type")
	}
	var req SubscribeRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return err
	}

	worker.subscribed = true

	result := []interface{}{
		[]interface{}{"mining.notify", ""},
		"",
		4,
	}

	resp := StratumMessage{
		ID:     msg.ID,
		Method: "mining.subscribe",
		Result: result,
	}

	return s.sendJSON(worker.conn, resp)
}

func (s *StratumServer) handleAuthorize(worker *StratumWorker, msg *StratumMessage) error {
	var params json.RawMessage
	if p, ok := msg.Params.(json.RawMessage); ok {
		params = p
	} else if p, ok := msg.Params.([]byte); ok {
		params = p
	} else {
		return fmt.Errorf("invalid params type")
	}
	var paramsSlice []string
	if err := json.Unmarshal(params, &paramsSlice); err != nil {
		return err
	}

	if len(paramsSlice) < 2 {
		return fmt.Errorf("missing username/password")
	}

	worker.user = paramsSlice[0]
	worker.password = paramsSlice[1]

	resp := StratumMessage{
		ID:     msg.ID,
		Method: "mining.authorize",
		Result: true,
	}

	return s.sendJSON(worker.conn, resp)
}

func (s *StratumServer) handleSubmit(worker *StratumWorker, msg *StratumMessage) error {
	var params json.RawMessage
	if p, ok := msg.Params.(json.RawMessage); ok {
		params = p
	} else if p, ok := msg.Params.([]byte); ok {
		params = p
	} else {
		return fmt.Errorf("invalid params type")
	}
	var req SubmitRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return err
	}

	worker.lastShare = time.Now()
	worker.shares.Add(1)

	nonce, ok := parseHexUint64(req.Nonce)
	if !ok {
		s.invalidShares.Add(1)
		return fmt.Errorf("invalid nonce")
	}

	ntime, ok := parseHexUint64(req.NTime)
	if !ok {
		s.invalidShares.Add(1)
		return fmt.Errorf("invalid ntime")
	}

	if !s.verifyShare(worker, req.JobID, req.ExtraNonce2, nonce, ntime) {
		s.invalidShares.Add(1)
		return nil
	}

	s.validShares.Add(1)

	submitted, err := s.miner.SubmitShare(req.JobID, req.ExtraNonce2, nonce, ntime)
	if err != nil {
		s.invalidShares.Add(1)
	} else if submitted {
		s.acceptedShares.Add(1)
		log.Printf("Worker %s found a block!", worker.user)
	}

	resp := StratumMessage{
		ID:     msg.ID,
		Method: "mining.submit",
		Result: true,
	}

	return s.sendJSON(worker.conn, resp)
}

func (s *StratumServer) handleGetTransactions(worker *StratumWorker, msg *StratumMessage) error {
	resp := StratumMessage{
		ID:     msg.ID,
		Method: "mining.get_transactions",
		Result: []interface{}{},
	}
	return s.sendJSON(worker.conn, resp)
}

func (s *StratumServer) handleExtranonceSubscribe(worker *StratumWorker, msg *StratumMessage) error {
	resp := StratumMessage{
		ID:     msg.ID,
		Method: "mining.extranonce",
		Result: nil,
	}
	return s.sendJSON(worker.conn, resp)
}

func (s *StratumServer) verifyShare(worker *StratumWorker, jobID, extraNonce2 string, nonce, ntime uint64) bool {
	job := s.getJob(jobID)
	if job == nil {
		return false
	}

	diff := worker.diff.Load()
	if diff == 0 {
		diff = 1
	}

	prevHashBytes, err := hex.DecodeString(job.PrevHash)
	if err != nil || len(prevHashBytes) != 32 {
		return false
	}

	coinbase1Bytes, err := hex.DecodeString(job.Coinbase1)
	if err != nil {
		return false
	}
	coinbase2Bytes, err := hex.DecodeString(job.Coinbase2)
	if err != nil {
		return false
	}

	coinbase := append(coinbase1Bytes, coinbase2Bytes...)
	coinbase = append(coinbase, []byte(extraNonce2)...)

	merkleRoot := coinbase
	for _, branch := range job.MerkleBranch {
		branchBytes, err := hex.DecodeString(branch)
		if err != nil {
			return false
		}
		combined := append(merkleRoot, branchBytes...)
		h := sha256.Sum256(combined)
		merkleRoot = h[:]
	}

	versionBytes, ok := parseHexUint64(job.Version)
	if !ok {
		return false
	}
	nbitsBytes, ok := parseHexUint64(job.NBits)
	if !ok {
		return false
	}

	header := make([]byte, 0)
	header = append(header, varintEncode(versionBytes)...)
	header = append(header, prevHashBytes...)
	header = append(header, merkleRoot[:]...)
	header = append(header, varintEncode(ntime)...)
	header = append(header, varintEncode(nbitsBytes)...)
	header = append(header, varintEncode(nonce)...)

	hash := sha256.Sum256(header)
	hashInt := new(big.Int).SetBytes(hash[:])

	target := new(big.Int).Lsh(big.NewInt(1), uint(256-diff))

	return hashInt.Cmp(target) < 0
}

func varintEncode(n uint64) []byte {
	if n < 0xfd {
		return []byte{byte(n)}
	} else if n <= 0xffff {
		return []byte{0xfd, byte(n), byte(n >> 8)}
	} else if n <= 0xffffffff {
		return []byte{0xfe, byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)}
	}
	return []byte{0xff, byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24), byte(n >> 32), byte(n >> 40), byte(n >> 48), byte(n >> 56)}
}

func (s *StratumServer) sendJSON(conn net.Conn, msg StratumMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}

func (s *StratumServer) sendError(conn net.Conn, id interface{}, code int, message string) {
	resp := StratumMessage{
		ID:    id,
		Error: []interface{}{code, message},
	}
	s.sendJSON(conn, resp)
}

func (s *StratumServer) BroadcastNewJob(job *StratumJob) {
	msg := StratumMessage{
		Method: "mining.notify",
		Params: MiningNotify{
			JobID:        job.ID,
			PrevHash:     job.PrevHash,
			Coinbase1:    job.Coinbase1,
			Coinbase2:    job.Coinbase2,
			MerkleBranch: job.MerkleBranch,
			Version:      job.Version,
			NBits:        job.NBits,
			NTime:        job.NTime,
			CleanJobs:    job.CleanJobs,
		},
	}

	data, _ := json.Marshal(msg)
	data = append(data, '\n')

	s.workers.Range(func(key, value any) bool {
		worker := value.(*StratumWorker)
		worker.conn.Write(data)
		return true
	})
}

func (s *StratumServer) Stop() {
	close(s.stopCh)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}

func (s *StratumServer) Stats() StratumStats {
	return StratumStats{
		Workers:        s.workerCount(),
		ValidShares:    s.validShares.Load(),
		InvalidShares:  s.invalidShares.Load(),
		AcceptedShares: s.acceptedShares.Load(),
	}
}

func (s *StratumServer) getJob(jobID string) *StratumJob {
	if job, ok := s.jobs.Load(jobID); ok {
		return job.(*StratumJob)
	}
	return nil
}

func (s *StratumServer) setJob(job *StratumJob) {
	s.jobs.Store(job.ID, job)
	s.currentJob.Store(job)
}

func (s *StratumServer) SubmitShare(worker *StratumWorker, jobID, extraNonce2 string, nonce, ntime uint64) (bool, error) {
	job := s.getJob(jobID)
	if job == nil {
		return false, fmt.Errorf("job not found: %s", jobID)
	}

	diff := worker.diff.Load()
	if diff == 0 {
		diff = 1
	}

	if !s.verifyShare(worker, jobID, extraNonce2, nonce, ntime) {
		s.invalidShares.Add(1)
		return false, nil
	}

	submitted, err := s.miner.SubmitShare(jobID, extraNonce2, nonce, ntime)
	if err != nil {
		s.invalidShares.Add(1)
		return false, err
	}

	s.validShares.Add(1)
	if submitted {
		s.acceptedShares.Add(1)
	}

	return submitted, nil
}

func (s *StratumServer) BroadcastJob(job *StratumJob) {
	s.setJob(job)
	s.BroadcastNewJob(job)
}

func (s *StratumServer) workerCount() int {
	count := 0
	s.workers.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

type StratumStats struct {
	Workers        int   `json:"workers"`
	ValidShares    int64 `json:"valid_shares"`
	InvalidShares  int64 `json:"invalid_shares"`
	AcceptedShares int64 `json:"accepted_shares"`
}

type StratumJob struct {
	ID           string
	PrevHash     string
	Coinbase1    string
	Coinbase2    string
	MerkleBranch []string
	Version      string
	NBits        string
	NTime        string
	CleanJobs    bool
}

func parseHexUint64(s string) (uint64, bool) {
	s = trimPrefix(s, "0x")
	n, err := hex.DecodeString(s)
	if err != nil {
		return 0, false
	}
	return new(big.Int).SetBytes(n).Uint64(), true
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
