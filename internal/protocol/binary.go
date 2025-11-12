package protocol

import (
	"encoding/binary"
	"fmt"
)

// Binary protocol format for maximum performance
//
// Frame format:
// [1 byte: OpCode][4 bytes: PayloadLength][Payload]
//
// OpCode values:
//   0x01 = SET
//   0x02 = GET
//   0x03 = DEL
//   0x04 = EXISTS
//   0x05 = INCR
//   0x06 = DECR
//   0x10 = MSET (batch set)
//   0x11 = MGET (batch get)
//   0x12 = MDEL (batch delete)
//
// Payload formats:
//
// SET: [2 bytes: keyLen][key][4 bytes: valueLen][value]
// GET: [2 bytes: keyLen][key]
// DEL: [2 bytes: keyLen][key]
// EXISTS: [2 bytes: keyLen][key]
// INCR/DECR: [2 bytes: keyLen][key]
//
// MSET: [2 bytes: count][for each: [2 bytes: keyLen][key][4 bytes: valueLen][value]]
// MGET: [2 bytes: count][for each: [2 bytes: keyLen][key]]
// MDEL: [2 bytes: count][for each: [2 bytes: keyLen][key]]
//
// Response format:
// [1 byte: Status][4 bytes: PayloadLength][Payload]
//
// Status values:
//   0x00 = OK
//   0x01 = Error
//   0x02 = NotFound
//   0x03 = MultiValue (for batch responses)

const (
	// Operation codes
	OpSet    byte = 0x01
	OpGet    byte = 0x02
	OpDel    byte = 0x03
	OpExists byte = 0x04
	OpIncr   byte = 0x05
	OpDecr   byte = 0x06
	OpMSet   byte = 0x10
	OpMGet   byte = 0x11
	OpMDel   byte = 0x12

	// Status codes
	StatusOK         byte = 0x00
	StatusError      byte = 0x01
	StatusNotFound   byte = 0x02
	StatusMultiValue byte = 0x03

	// Protocol constants
	MaxKeyLen    = 65535     // 2 bytes
	MaxValueLen  = 1 << 30   // 1GB
	MaxBatchSize = 10000     // Maximum keys per batch
)

// Request represents a parsed binary request
type Request struct {
	OpCode byte
	Key    string
	Value  []byte
	Keys   []string
	Values [][]byte
}

// Response represents a binary response
type Response struct {
	Status byte
	Value  []byte
	Values [][]byte
	Error  string
}

// EncodeSetRequest encodes a SET request
func EncodeSetRequest(key string, value []byte) []byte {
	keyLen := len(key)
	valueLen := len(value)

	// Calculate total size: 1 (opcode) + 4 (payload len) + 2 (key len) + key + 4 (value len) + value
	totalSize := 1 + 4 + 2 + keyLen + 4 + valueLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpSet
	pos++

	// Payload length
	payloadLen := 2 + keyLen + 4 + valueLen
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	// Key
	binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
	pos += 2
	copy(buf[pos:], key)
	pos += keyLen

	// Value
	binary.BigEndian.PutUint32(buf[pos:], uint32(valueLen))
	pos += 4
	copy(buf[pos:], value)

	return buf
}

// EncodeGetRequest encodes a GET request
func EncodeGetRequest(key string) []byte {
	keyLen := len(key)
	totalSize := 1 + 4 + 2 + keyLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpGet
	pos++

	payloadLen := 2 + keyLen
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
	pos += 2
	copy(buf[pos:], key)

	return buf
}

// EncodeDeleteRequest encodes a DEL request
func EncodeDeleteRequest(key string) []byte {
	return encodeSimpleRequest(OpDel, key)
}

// EncodeExistsRequest encodes an EXISTS request
func EncodeExistsRequest(key string) []byte {
	return encodeSimpleRequest(OpExists, key)
}

// EncodeIncrRequest encodes an INCR request
func EncodeIncrRequest(key string) []byte {
	return encodeSimpleRequest(OpIncr, key)
}

// EncodeDecrRequest encodes a DECR request
func EncodeDecrRequest(key string) []byte {
	return encodeSimpleRequest(OpDecr, key)
}

func encodeSimpleRequest(opCode byte, key string) []byte {
	keyLen := len(key)
	totalSize := 1 + 4 + 2 + keyLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = opCode
	pos++

	payloadLen := 2 + keyLen
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
	pos += 2
	copy(buf[pos:], key)

	return buf
}

// EncodeMSetRequest encodes a batch SET request
func EncodeMSetRequest(keys []string, values [][]byte) []byte {
	if len(keys) != len(values) {
		return nil
	}

	// Calculate total size
	totalSize := 1 + 4 + 2 // opcode + payload len + count
	for i := range keys {
		totalSize += 2 + len(keys[i]) + 4 + len(values[i])
	}

	buf := make([]byte, totalSize)
	pos := 0

	buf[pos] = OpMSet
	pos++

	// Payload length (everything after the first 5 bytes)
	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	// Count
	binary.BigEndian.PutUint16(buf[pos:], uint16(len(keys)))
	pos += 2

	// Key-value pairs
	for i := range keys {
		// Key
		keyLen := len(keys[i])
		binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
		pos += 2
		copy(buf[pos:], keys[i])
		pos += keyLen

		// Value
		valueLen := len(values[i])
		binary.BigEndian.PutUint32(buf[pos:], uint32(valueLen))
		pos += 4
		copy(buf[pos:], values[i])
		pos += valueLen
	}

	return buf
}

// EncodeMGetRequest encodes a batch GET request
func EncodeMGetRequest(keys []string) []byte {
	totalSize := 1 + 4 + 2
	for _, key := range keys {
		totalSize += 2 + len(key)
	}

	buf := make([]byte, totalSize)
	pos := 0

	buf[pos] = OpMGet
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(len(keys)))
	pos += 2

	for _, key := range keys {
		keyLen := len(key)
		binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
		pos += 2
		copy(buf[pos:], key)
		pos += keyLen
	}

	return buf
}

// EncodeMDeleteRequest encodes a batch DELETE request
func EncodeMDeleteRequest(keys []string) []byte {
	totalSize := 1 + 4 + 2
	for _, key := range keys {
		totalSize += 2 + len(key)
	}

	buf := make([]byte, totalSize)
	pos := 0

	buf[pos] = OpMDel
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(len(keys)))
	pos += 2

	for _, key := range keys {
		keyLen := len(key)
		binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
		pos += 2
		copy(buf[pos:], key)
		pos += keyLen
	}

	return buf
}

// DecodeRequest parses a binary request
func DecodeRequest(data []byte) (*Request, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("request too short")
	}

	req := &Request{}
	req.OpCode = data[0]
	payloadLen := binary.BigEndian.Uint32(data[1:5])

	if len(data) < int(5+payloadLen) {
		return nil, fmt.Errorf("incomplete request")
	}

	payload := data[5 : 5+payloadLen]

	switch req.OpCode {
	case OpSet:
		return decodeSetRequest(payload)
	case OpGet, OpDel, OpExists, OpIncr, OpDecr:
		return decodeSimpleRequest(req.OpCode, payload)
	case OpMSet:
		return decodeMSetRequest(payload)
	case OpMGet, OpMDel:
		return decodeMGetRequest(req.OpCode, payload)
	default:
		return nil, fmt.Errorf("unknown opcode: %d", req.OpCode)
	}
}

func decodeSetRequest(payload []byte) (*Request, error) {
	if len(payload) < 6 {
		return nil, fmt.Errorf("invalid SET payload")
	}

	req := &Request{OpCode: OpSet}
	pos := 0

	keyLen := binary.BigEndian.Uint16(payload[pos:])
	pos += 2

	if len(payload) < int(pos+int(keyLen)+4) {
		return nil, fmt.Errorf("invalid SET payload")
	}

	req.Key = string(payload[pos : pos+int(keyLen)])
	pos += int(keyLen)

	valueLen := binary.BigEndian.Uint32(payload[pos:])
	pos += 4

	if len(payload) < int(pos+int(valueLen)) {
		return nil, fmt.Errorf("invalid SET payload")
	}

	req.Value = payload[pos : pos+int(valueLen)]

	return req, nil
}

func decodeSimpleRequest(opCode byte, payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid payload")
	}

	req := &Request{OpCode: opCode}
	keyLen := binary.BigEndian.Uint16(payload[0:2])

	if len(payload) < int(2+keyLen) {
		return nil, fmt.Errorf("invalid payload")
	}

	req.Key = string(payload[2 : 2+keyLen])
	return req, nil
}

func decodeMSetRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid MSET payload")
	}

	req := &Request{OpCode: OpMSet}
	count := binary.BigEndian.Uint16(payload[0:2])
	pos := 2

	req.Keys = make([]string, 0, count)
	req.Values = make([][]byte, 0, count)

	for i := 0; i < int(count); i++ {
		if len(payload) < pos+2 {
			return nil, fmt.Errorf("invalid MSET payload")
		}

		keyLen := binary.BigEndian.Uint16(payload[pos:])
		pos += 2

		if len(payload) < pos+int(keyLen)+4 {
			return nil, fmt.Errorf("invalid MSET payload")
		}

		key := string(payload[pos : pos+int(keyLen)])
		pos += int(keyLen)

		valueLen := binary.BigEndian.Uint32(payload[pos:])
		pos += 4

		if len(payload) < pos+int(valueLen) {
			return nil, fmt.Errorf("invalid MSET payload")
		}

		value := payload[pos : pos+int(valueLen)]
		pos += int(valueLen)

		req.Keys = append(req.Keys, key)
		req.Values = append(req.Values, value)
	}

	return req, nil
}

func decodeMGetRequest(opCode byte, payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid batch payload")
	}

	req := &Request{OpCode: opCode}
	count := binary.BigEndian.Uint16(payload[0:2])
	pos := 2

	req.Keys = make([]string, 0, count)

	for i := 0; i < int(count); i++ {
		if len(payload) < pos+2 {
			return nil, fmt.Errorf("invalid batch payload")
		}

		keyLen := binary.BigEndian.Uint16(payload[pos:])
		pos += 2

		if len(payload) < pos+int(keyLen) {
			return nil, fmt.Errorf("invalid batch payload")
		}

		key := string(payload[pos : pos+int(keyLen)])
		pos += int(keyLen)

		req.Keys = append(req.Keys, key)
	}

	return req, nil
}

// EncodeOKResponse encodes a success response
func EncodeOKResponse() []byte {
	buf := make([]byte, 5)
	buf[0] = StatusOK
	binary.BigEndian.PutUint32(buf[1:], 0) // No payload
	return buf
}

// EncodeValueResponse encodes a single value response
func EncodeValueResponse(value []byte) []byte {
	totalSize := 5 + len(value)
	buf := make([]byte, totalSize)

	buf[0] = StatusOK
	binary.BigEndian.PutUint32(buf[1:], uint32(len(value)))
	copy(buf[5:], value)

	return buf
}

// EncodeMultiValueResponse encodes a batch response
func EncodeMultiValueResponse(values [][]byte) []byte {
	// Calculate size: status + payload len + count + (len + value) for each
	totalSize := 5 + 2
	for _, v := range values {
		totalSize += 4 + len(v)
	}

	buf := make([]byte, totalSize)
	pos := 0

	buf[pos] = StatusMultiValue
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(len(values)))
	pos += 2

	for _, v := range values {
		binary.BigEndian.PutUint32(buf[pos:], uint32(len(v)))
		pos += 4
		copy(buf[pos:], v)
		pos += len(v)
	}

	return buf
}

// EncodeErrorResponse encodes an error response
func EncodeErrorResponse(err error) []byte {
	errMsg := err.Error()
	totalSize := 5 + len(errMsg)
	buf := make([]byte, totalSize)

	buf[0] = StatusError
	binary.BigEndian.PutUint32(buf[1:], uint32(len(errMsg)))
	copy(buf[5:], errMsg)

	return buf
}

// DecodeResponse parses a binary response
func DecodeResponse(data []byte) (*Response, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("response too short")
	}

	resp := &Response{}
	resp.Status = data[0]
	payloadLen := binary.BigEndian.Uint32(data[1:5])

	if len(data) < int(5+payloadLen) {
		return nil, fmt.Errorf("incomplete response")
	}

	payload := data[5 : 5+payloadLen]

	switch resp.Status {
	case StatusOK:
		if len(payload) > 0 {
			resp.Value = payload
		}
	case StatusError:
		resp.Error = string(payload)
	case StatusMultiValue:
		return decodeMultiValueResponse(payload)
	case StatusNotFound:
		// No payload
	default:
		return nil, fmt.Errorf("unknown status: %d", resp.Status)
	}

	return resp, nil
}

func decodeMultiValueResponse(payload []byte) (*Response, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid multi-value response")
	}

	resp := &Response{Status: StatusMultiValue}
	count := binary.BigEndian.Uint16(payload[0:2])
	pos := 2

	resp.Values = make([][]byte, 0, count)

	for i := 0; i < int(count); i++ {
		if len(payload) < pos+4 {
			return nil, fmt.Errorf("invalid multi-value response")
		}

		valueLen := binary.BigEndian.Uint32(payload[pos:])
		pos += 4

		if len(payload) < pos+int(valueLen) {
			return nil, fmt.Errorf("invalid multi-value response")
		}

		value := payload[pos : pos+int(valueLen)]
		pos += int(valueLen)

		resp.Values = append(resp.Values, value)
	}

	return resp, nil
}
