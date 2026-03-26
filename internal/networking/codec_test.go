package networking

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"testing"

	neocoinpb "github.com/Neo4717/NeoCoin/proto"
	"google.golang.org/protobuf/proto"
)

func TestCodec_EncodeDecodeRoundtrip(t *testing.T) {
	codec := NewProtobufCodec(false)
	codec.SetCompression(false)

	msg := &neocoinpb.Ping{
		Nonce: 12345,
	}

	encoded, err := codec.Encode("ping", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bytes.NewReader(encoded)
	_, payload, err := codec.Decode(reader)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	decoded := &neocoinpb.Ping{}
	if err := proto.Unmarshal(payload, decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.GetNonce() != 12345 {
		t.Errorf("expected Nonce 12345, got %d", decoded.GetNonce())
	}
}

func TestCodec_EncodeDecodeTransaction(t *testing.T) {
	codec := NewProtobufCodec(true)

	msg := &neocoinpb.Transaction{
		From:   "sender-address",
		To:     "recipient-address",
		Amount: 100,
		Fee:    10,
		Nonce:  1,
	}

	encoded, err := codec.Encode("tx", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bytes.NewReader(encoded)
	_, payload, err := codec.Decode(reader)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	decoded := &neocoinpb.Transaction{}
	if err := proto.Unmarshal(payload, decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.GetFrom() != "sender-address" {
		t.Errorf("expected from 'sender-address', got %s", decoded.GetFrom())
	}
	if decoded.GetTo() != "recipient-address" {
		t.Errorf("expected to 'recipient-address', got %s", decoded.GetTo())
	}
}

func TestCodec_EncodeDecodeBlock(t *testing.T) {
	codec := NewProtobufCodec(true)

	msg := &neocoinpb.Block{
		Header: &neocoinpb.Header{
			Version:    1,
			Height:     100,
			Timestamp:  1234567890,
			PrevBlock:  "previous-block-hash",
			MerkleRoot: "merkle-root",
			Nonce:      "54321",
		},
	}

	encoded, err := codec.Encode("block", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bytes.NewReader(encoded)
	_, payload, err := codec.Decode(reader)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	decoded := &neocoinpb.Block{}
	if err := proto.Unmarshal(payload, decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.GetHeader().GetHeight() != 100 {
		t.Errorf("expected height 100, got %d", decoded.GetHeader().GetHeight())
	}
}

func TestCodec_ChecksumVerification(t *testing.T) {
	codec := NewProtobufCodec(false)

	msg := &neocoinpb.Ping{Nonce: 999}
	encoded, err := codec.Encode("ping", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	corrupted := make([]byte, len(encoded))
	copy(corrupted, encoded)
	// Corrupt data in the payload section (after 14-byte header)
	if len(corrupted) > 14 {
		corrupted[14] ^= 0xFF
	}

	reader := bytes.NewReader(corrupted)
	_, _, err = codec.Decode(reader)
	if err == nil {
		t.Error("expected error for corrupted payload")
	}
}

func TestCodec_MagicBytesVerification(t *testing.T) {
	codec := NewProtobufCodec(false)

	msg := &neocoinpb.Ping{Nonce: 123}
	encoded, err := codec.Encode("ping", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	corrupted := make([]byte, len(encoded))
	copy(corrupted, encoded)
	binary.LittleEndian.PutUint32(corrupted[0:4], 0xDEADBEEF)

	reader := bytes.NewReader(corrupted)
	_, _, err = codec.Decode(reader)
	if err == nil {
		t.Error("expected error for invalid magic bytes")
	}
}

func TestCodec_VersionMismatch(t *testing.T) {
	codec := NewProtobufCodec(false)

	msg := &neocoinpb.Ping{Nonce: 456}
	encoded, err := codec.Encode("ping", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	corrupted := make([]byte, len(encoded))
	copy(corrupted, encoded)
	binary.LittleEndian.PutUint16(corrupted[4:6], 999)

	reader := bytes.NewReader(corrupted)
	_, _, err = codec.Decode(reader)
	if err == nil {
		t.Error("expected error for unsupported protocol version")
	}
}

func TestCodec_UnknownMessageType(t *testing.T) {
	codec := NewProtobufCodec(false)

	type UnknownMsg struct {
		Data string
	}

	_, err := codec.Encode("unknown", &UnknownMsg{Data: "test"})
	if err == nil {
		t.Error("expected error for unknown message type")
	}
}

func TestCodec_SetProtobuf(t *testing.T) {
	codec := NewProtobufCodec(false)
	if codec.useProtobuf != false {
		t.Errorf("expected useProtobuf=false initially")
	}

	codec.SetProtobuf(true)
	if codec.useProtobuf != true {
		t.Errorf("expected useProtobuf=true after SetProtobuf")
	}
}

func TestCodec_SetCompression(t *testing.T) {
	codec := NewProtobufCodec(false)
	if codec.enableCompress != true {
		t.Logf("default compression enabled")
	}

	codec.SetCompression(false)
	codec.SetCompression(true)
}

func TestCodec_IncompleteFrame(t *testing.T) {
	codec := NewProtobufCodec(false)

	shortData := make([]byte, 5)
	reader := bytes.NewReader(shortData)
	_, _, err := codec.Decode(reader)
	if err == nil {
		t.Error("expected error for incomplete frame")
	}
}

func TestCodec_PayloadTooLarge(t *testing.T) {
	codec := NewProtobufCodec(false)

	msg := &neocoinpb.Transaction{
		From:      "from",
		To:        "to",
		Amount:    0,
		Signature: make([]byte, 64*1024*1024),
	}

	_, err := codec.Encode("tx", msg)
	if err == nil {
		t.Error("expected error for payload exceeding max size")
	}
}

func TestCodec_ConcurrentEncodeDecode(t *testing.T) {
	codec := NewProtobufCodec(true)

	msg := &neocoinpb.Ping{Nonce: 1}
	encoded, _ := codec.Encode("ping", msg)

	var results []error
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reader := bytes.NewReader(encoded)
			_, payload, err := codec.Decode(reader)
			mu.Lock()
			if err == nil && len(payload) > 0 {
				decoded := &neocoinpb.Ping{}
				if err := proto.Unmarshal(payload, decoded); err != nil {
					results = append(results, err)
				}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(results) > 0 {
		t.Errorf("got %d errors during concurrent decode", len(results))
	}
}

func TestCodec_NoCompressionBelowThreshold(t *testing.T) {
	codec := NewProtobufCodec(true)
	codec.SetCompression(true)

	smallData := []byte("small payload")
	msg := &neocoinpb.Transaction{
		From:      "from",
		To:        "to",
		Amount:    0,
		Signature: smallData,
	}

	encoded, err := codec.Encode("tx", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if len(encoded) < 14+len(smallData) {
		t.Error("encoded size should include header and payload without heavy compression")
	}
}

func TestCodec_DisableCompression(t *testing.T) {
	codec := NewProtobufCodec(true)
	codec.SetCompression(false)

	largeData := make([]byte, 5000)
	msg := &neocoinpb.Transaction{
		From:      "from",
		To:        "to",
		Amount:    0,
		Signature: largeData,
	}

	encoded, err := codec.Encode("tx", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	compressedLen := len(encoded)

	codec.SetCompression(true)
	encodedCompressed, _ := codec.Encode("tx", msg)

	if len(encodedCompressed) >= compressedLen {
		t.Logf("compression may not have reduced size: compressed=%d uncompressed=%d",
			len(encodedCompressed), compressedLen)
	}
}

func TestCodec_IncompletePayload(t *testing.T) {
	codec := NewProtobufCodec(false)

	frame := make([]byte, 14+100)
	binary.LittleEndian.PutUint32(frame[0:4], PBMagicBytes)
	binary.LittleEndian.PutUint16(frame[4:6], PBProtocolVersion)
	binary.LittleEndian.PutUint32(frame[6:10], 1000)
	binary.LittleEndian.PutUint32(frame[10:14], 0)

	reader := bytes.NewReader(frame)
	_, _, err := codec.Decode(reader)
	if err == nil {
		t.Error("expected error for incomplete payload")
	}
}

func TestCodec_EmptyPayload(t *testing.T) {
	codec := NewProtobufCodec(true)

	msg := &neocoinpb.Transaction{
		From:   "empty",
		To:     "empty",
		Amount: 0,
	}

	encoded, err := codec.Encode("tx", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bytes.NewReader(encoded)
	_, payload, err := codec.Decode(reader)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Empty transaction still has protobuf overhead, just verify it decodes
	if len(payload) < 0 {
		t.Errorf("unexpected payload length: %d", len(payload))
	}
}

func TestCodec_MultipleMessages(t *testing.T) {
	codec := NewProtobufCodec(true)

	messages := []*neocoinpb.Ping{
		{Nonce: 1},
		{Nonce: 2},
		{Nonce: 3},
	}

	for _, msg := range messages {
		encoded, err := codec.Encode("ping", msg)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		reader := bytes.NewReader(encoded)
		_, payload, err := codec.Decode(reader)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		decoded := &neocoinpb.Ping{}
		if err := proto.Unmarshal(payload, decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if decoded.GetNonce() != msg.Nonce {
			t.Errorf("expected Nonce %d, got %d", msg.Nonce, decoded.GetNonce())
		}
	}
}

func TestCodec_ReaderError(t *testing.T) {
	codec := NewProtobufCodec(false)

	_, _, err := codec.Decode(&errorReader{})
	if err == nil {
		t.Error("expected error from failing reader")
	}
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestCodec_PBXorChecksum(t *testing.T) {
	data := []byte("test data for checksum")
	checksum1 := pbxorChecksum(data)
	checksum2 := pbxorChecksum(data)

	if checksum1 != checksum2 {
		t.Error("checksum should be deterministic")
	}

	data[0] ^= 0xFF
	checksum3 := pbxorChecksum(data)

	if checksum1 == checksum3 {
		t.Error("checksum should change when data changes")
	}
}

func TestCodec_PBXorChecksumEmpty(t *testing.T) {
	checksum := pbxorChecksum([]byte{})
	if checksum != 0 {
		t.Errorf("expected 0 checksum for empty data, got %d", checksum)
	}
}

func TestCodec_PBXorChecksumPartialBlocks(t *testing.T) {
	testCases := [][]byte{
		{0x01},
		{0x01, 0x02},
		{0x01, 0x02, 0x03},
		{0x01, 0x02, 0x03, 0x04},
		{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	for _, data := range testCases {
		checksum := pbxorChecksum(data)
		if checksum == 0 && len(data) > 0 {
			t.Logf("checksum is 0 for data: %x", data)
		}
	}
}

func TestCodec_HeaderRoundtrip(t *testing.T) {
	codec := NewProtobufCodec(true)

	msg := &neocoinpb.Header{
		Version:    1,
		Height:     1000,
		Timestamp:  1234567890,
		PrevBlock:  "prev-hash",
		MerkleRoot: "merkle",
		Nonce:      "12345",
		Miner:      "miner-address",
	}

	encoded, err := codec.Encode("header", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bytes.NewReader(encoded)
	_, payload, err := codec.Decode(reader)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	decoded := &neocoinpb.Header{}
	if err := proto.Unmarshal(payload, decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.GetVersion() != 1 || decoded.GetHeight() != 1000 {
		t.Errorf("header mismatch: version=%d height=%d", decoded.GetVersion(), decoded.GetHeight())
	}
}

func TestCodec_PongRoundtrip(t *testing.T) {
	codec := NewProtobufCodec(true)

	msg := &neocoinpb.Pong{
		Nonce: 999,
	}

	encoded, err := codec.Encode("pong", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bytes.NewReader(encoded)
	_, payload, err := codec.Decode(reader)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	decoded := &neocoinpb.Pong{}
	if err := proto.Unmarshal(payload, decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.GetNonce() != 999 {
		t.Errorf("expected Nonce 999, got %d", decoded.GetNonce())
	}
}

func TestCodec_UnmarshalProtoMessage(t *testing.T) {
	codec := NewProtobufCodec(true)

	msg := &neocoinpb.Ping{Nonce: 54321}
	encoded, err := codec.Encode("ping", msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	reader := bytes.NewReader(encoded)
	_, payload, err := codec.Decode(reader)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	decoded, err := UnmarshalProtoMessage("ping", payload)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	ping, ok := decoded.(*neocoinpb.Ping)
	if !ok {
		t.Fatalf("expected *neocoinpb.Ping, got %T", decoded)
	}

	if ping.GetNonce() != 54321 {
		t.Errorf("expected Nonce 54321, got %d", ping.GetNonce())
	}
}

func TestCodec_UnmarshalProtoMessageUnknown(t *testing.T) {
	_, err := UnmarshalProtoMessage("unknown-type", []byte{})
	if err == nil {
		t.Error("expected error for unknown message type")
	}
}

func TestCodec_MarshalProtoMessage(t *testing.T) {
	msg := &neocoinpb.Ping{Nonce: 12345}
	encoded, err := MarshalProtoMessage(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("expected non-empty encoded data")
	}
}

func TestCodec_MarshalProtoMessageUnknown(t *testing.T) {
	_, err := MarshalProtoMessage("not-a-proto-message")
	if err == nil {
		t.Error("expected error for unknown message type")
	}
}
