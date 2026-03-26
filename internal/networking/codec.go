package networking

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"google.golang.org/protobuf/proto"

	neocoinpb "github.com/Neo4717/NeoCoin/proto"
)

const (
	PBMagicBytes           = 0x4E454F43
	PBMaxPayloadSize       = 32 * 1024 * 1024
	PBProtocolVersion      = 1
	PBCompressionThreshold = 1024
)

type ProtobufCodec struct {
	useProtobuf    bool
	enableCompress bool
	mu             sync.RWMutex
}

func NewProtobufCodec(useProtobuf bool) *ProtobufCodec {
	return &ProtobufCodec{
		useProtobuf:    useProtobuf,
		enableCompress: true,
	}
}

func (c *ProtobufCodec) SetProtobuf(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.useProtobuf = enabled
}

func (c *ProtobufCodec) SetCompression(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enableCompress = enabled
}

func (c *ProtobufCodec) Encode(msgType string, msg interface{}) ([]byte, error) {
	var payload []byte
	var err error

	switch m := msg.(type) {
	case *neocoinpb.Block:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Header:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Transaction:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Ping:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Pong:
		payload, err = proto.Marshal(m)
	case *neocoinpb.GetHeaders:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Headers:
		payload, err = proto.Marshal(m)
	case *neocoinpb.GetBlocks:
		payload, err = proto.Marshal(m)
	case *neocoinpb.GetTransactions:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Inv:
		payload, err = proto.Marshal(m)
	case *neocoinpb.GetData:
		payload, err = proto.Marshal(m)
	case *neocoinpb.NotFound:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Version:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Verack:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Reject:
		payload, err = proto.Marshal(m)
	case *neocoinpb.P2PMessage:
		payload, err = proto.Marshal(m)
	case *neocoinpb.Hello:
		payload, err = proto.Marshal(m)
	default:
		return nil, fmt.Errorf("unknown message type: %T", msg)
	}

	if err != nil {
		return nil, fmt.Errorf("proto marshal: %w", err)
	}

	if len(payload) > PBMaxPayloadSize {
		return nil, fmt.Errorf("payload too large: %d > %d", len(payload), PBMaxPayloadSize)
	}

	framed := c.frame(payload)

	c.mu.RLock()
	compress := c.enableCompress
	c.mu.RUnlock()

	if compress && len(framed) > PBCompressionThreshold {
		framed = c.compress(framed)
	}

	return framed, nil
}

func (c *ProtobufCodec) frame(payload []byte) []byte {
	buf := make([]byte, 14+len(payload))
	binary.LittleEndian.PutUint32(buf[0:4], PBMagicBytes)
	binary.LittleEndian.PutUint16(buf[4:6], PBProtocolVersion)
	binary.LittleEndian.PutUint32(buf[6:10], uint32(len(payload)))
	checksum := pbxorChecksum(payload)
	binary.LittleEndian.PutUint32(buf[10:14], checksum)
	copy(buf[14:], payload)
	return buf
}

func (c *ProtobufCodec) compress(data []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(1)
	gzData := compressGzip(data)
	buf.Write(gzData)
	return buf.Bytes()
}

func (c *ProtobufCodec) decompress(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return data, nil
	}
	if data[0] == 1 {
		return decompressGzip(data[1:])
	}
	return data, nil
}

func (c *ProtobufCodec) Decode(r io.Reader) (string, []byte, error) {
	header := make([]byte, 1)
	if _, err := io.ReadFull(r, header); err != nil {
		return "", nil, err
	}

	compressed := header[0] == 1
	var framed []byte

	if compressed {
		compressedData, err := io.ReadAll(r)
		if err != nil {
			return "", nil, err
		}
		decompressed, err := c.decompress(compressedData)
		if err != nil {
			return "", nil, err
		}
		framed = append([]byte{0}, decompressed...)
	} else {
		rest, err := io.ReadAll(r)
		if err != nil {
			return "", nil, err
		}
		framed = append(header, rest...)
	}

	if len(framed) < 14 {
		return "", nil, fmt.Errorf("incomplete frame")
	}

	magic := binary.LittleEndian.Uint32(framed[0:4])
	if magic != PBMagicBytes {
		return "", nil, fmt.Errorf("invalid magic: 0x%08x", magic)
	}

	version := binary.LittleEndian.Uint16(framed[4:6])
	if version != PBProtocolVersion {
		return "", nil, fmt.Errorf("unsupported protocol version: %d", version)
	}

	length := binary.LittleEndian.Uint32(framed[6:10])
	if length > PBMaxPayloadSize {
		return "", nil, fmt.Errorf("payload too large: %d", length)
	}

	checksum := binary.LittleEndian.Uint32(framed[10:14])

	if len(framed) < 14+int(length) {
		return "", nil, fmt.Errorf("incomplete payload")
	}

	payload := framed[14 : 14+length]

	if pbxorChecksum(payload) != checksum {
		return "", nil, fmt.Errorf("checksum mismatch")
	}

	return "", payload, nil
}

func pbxorChecksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		if i+4 <= len(data) {
			sum ^= binary.LittleEndian.Uint32(data[i : i+4])
		} else if i+3 <= len(data) {
			sum ^= uint32(data[i]) | uint32(data[i+1])<<8 | uint32(data[i+2])<<16
		} else if i+2 <= len(data) {
			sum ^= uint32(data[i]) | uint32(data[i+1])<<8
		} else {
			sum ^= uint32(data[i])
		}
	}
	return sum
}

func compressGzip(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()
	return buf.Bytes()
}

func decompressGzip(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
}

func MarshalProtoMessage(msg interface{}) ([]byte, error) {
	switch m := msg.(type) {
	case *neocoinpb.Block:
		return proto.Marshal(m)
	case *neocoinpb.Transaction:
		return proto.Marshal(m)
	case *neocoinpb.Header:
		return proto.Marshal(m)
	case *neocoinpb.Ping:
		return proto.Marshal(m)
	case *neocoinpb.Pong:
		return proto.Marshal(m)
	case *neocoinpb.GetHeaders:
		return proto.Marshal(m)
	case *neocoinpb.Headers:
		return proto.Marshal(m)
	case *neocoinpb.Inv:
		return proto.Marshal(m)
	case *neocoinpb.GetData:
		return proto.Marshal(m)
	case *neocoinpb.Version:
		return proto.Marshal(m)
	case *neocoinpb.Verack:
		return proto.Marshal(m)
	case *neocoinpb.Reject:
		return proto.Marshal(m)
	case *neocoinpb.GetBlocks:
		return proto.Marshal(m)
	case *neocoinpb.GetTransactions:
		return proto.Marshal(m)
	case *neocoinpb.NotFound:
		return proto.Marshal(m)
	case *neocoinpb.Hello:
		return proto.Marshal(m)
	default:
		return nil, fmt.Errorf("unknown message type: %T", msg)
	}
}

func UnmarshalProtoMessage(msgType string, data []byte) (interface{}, error) {
	switch msgType {
	case "block":
		msg := &neocoinpb.Block{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "tx":
		msg := &neocoinpb.Transaction{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "headers":
		msg := &neocoinpb.Headers{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "ping":
		msg := &neocoinpb.Ping{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "pong":
		msg := &neocoinpb.Pong{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "inv":
		msg := &neocoinpb.Inv{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "getheaders":
		msg := &neocoinpb.GetHeaders{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "version":
		msg := &neocoinpb.Version{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "verack":
		return &neocoinpb.Verack{}, nil
	case "reject":
		msg := &neocoinpb.Reject{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "getdata":
		msg := &neocoinpb.GetData{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "getblocks":
		msg := &neocoinpb.GetBlocks{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "notfound":
		msg := &neocoinpb.NotFound{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	case "hello":
		msg := &neocoinpb.Hello{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}
}
