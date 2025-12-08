package net

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

	// Queue operation codes
	OpQPush   byte = 0x20
	OpQPop    byte = 0x21
	OpQPeek   byte = 0x22
	OpQLen    byte = 0x23
	OpQClear  byte = 0x24
	OpQPopMsg byte = 0x25 // Pop with acknowledgment
	OpQAck    byte = 0x26 // Acknowledge message
	OpQNack   byte = 0x27 // Negative acknowledge (requeue)

	// Stream operation codes
	OpSPublish      byte = 0x30
	OpSConsume      byte = 0x31
	OpSCommit       byte = 0x32
	OpSCreateTopic  byte = 0x33
	OpSSubscribe    byte = 0x34
	OpSUnsubscribe  byte = 0x35
	OpSGetOffsets   byte = 0x36
	OpSPublishBatch byte = 0x37 // Stream Publish Batch

	// Document operation codes
	OpDocInsert byte = 0x40
	OpDocFind   byte = 0x41
	OpDocUpdate byte = 0x42
	OpDocDelete byte = 0x43
	OpDocIndex  byte = 0x44

	// Status codes
	StatusOK         byte = 0x00
	StatusError      byte = 0x01
	StatusNotFound   byte = 0x02
	StatusKeyExists  byte = 0x03
	StatusMultiValue byte = 0x04

	// Protocol constants
	MaxKeyLen    = 65535   // 2 bytes
	MaxBatchSize = 10000   // Max messages per batch
	MaxValueLen  = 1 << 30 // 1GB
)

// Request represents a parsed binary request
type Request struct {
	OpCode byte
	Key    string
	Value  []byte
	Keys   []string
	Values [][]byte

	// Queue fields
	QueueName         string
	ConsumerID        string
	MessageID         string
	VisibilityTimeout int64 // milliseconds
	Requeue           bool

	// Stream fields
	Topic       string
	Partition   int
	Group       string
	Consumer    string
	Count       int
	RetentionMs int64
	Offset      int64 // Added Offset for stream operations

	// Batch operations
	BatchMessages []*BatchMessage

	// DocStore fields
	Collection string
}

// BatchMessage represents a single message in a batch
type BatchMessage struct {
	Partition int
	Key       string
	Value     []byte
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

// EncodeQPushRequest encodes a QPUSH request
func EncodeQPushRequest(queueName string, value []byte) []byte {
	nameLen := len(queueName)
	valueLen := len(value)
	totalSize := 1 + 4 + 2 + nameLen + 4 + valueLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpQPush
	pos++

	payloadLen := 2 + nameLen + 4 + valueLen
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(nameLen))
	pos += 2
	copy(buf[pos:], queueName)
	pos += nameLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(valueLen))
	pos += 4
	copy(buf[pos:], value)

	return buf
}

// EncodeQPopRequest encodes a QPOP request
func EncodeQPopRequest(queueName string) []byte {
	return encodeSimpleRequest(OpQPop, queueName)
}

// EncodeQPeekRequest encodes a QPEEK request
func EncodeQPeekRequest(queueName string) []byte {
	return encodeSimpleRequest(OpQPeek, queueName)
}

// EncodeQLenRequest encodes a QLEN request
func EncodeQLenRequest(queueName string) []byte {
	return encodeSimpleRequest(OpQLen, queueName)
}

// EncodeQClearRequest encodes a QCLEAR request
func EncodeQClearRequest(queueName string) []byte {
	return encodeSimpleRequest(OpQClear, queueName)
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
	case OpQPush:
		return decodeQPushRequest(payload)
	case OpQPop, OpQPeek, OpQLen, OpQClear:
		return decodeSimpleRequest(req.OpCode, payload)
	case OpSPublish:
		return decodeSPublishRequest(payload)
	case OpSConsume:
		return decodeSConsumeRequest(payload)
	case OpSCommit:
		return decodeSCommitRequest(payload)
	case OpSCreateTopic:
		return decodeSCreateTopicRequest(payload)
	case OpSPublishBatch:
		return decodeSPublishBatchRequest(payload)
	case OpSSubscribe:
		return decodeSSubscribeRequest(payload)
	case OpSUnsubscribe:
		return decodeSUnsubscribeRequest(payload)
	case OpDocInsert:
		return decodeDocInsertRequest(payload)
	case OpDocFind:
		return decodeDocFindRequest(payload)
	case OpDocUpdate:
		return decodeDocUpdateRequest(payload)
	case OpDocDelete:
		return decodeDocDeleteRequest(payload)
	case OpDocIndex:
		return decodeDocIndexRequest(payload)
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

func decodeQPushRequest(payload []byte) (*Request, error) {
	if len(payload) < 6 {
		return nil, fmt.Errorf("invalid QPUSH payload")
	}

	req := &Request{OpCode: OpQPush}
	pos := 0

	nameLen := binary.BigEndian.Uint16(payload[pos:])
	pos += 2

	if len(payload) < int(pos+int(nameLen)+4) {
		return nil, fmt.Errorf("invalid QPUSH payload")
	}

	req.Key = string(payload[pos : pos+int(nameLen)])
	pos += int(nameLen)

	valueLen := binary.BigEndian.Uint32(payload[pos:])
	pos += 4

	if len(payload) < int(pos+int(valueLen)) {
		return nil, fmt.Errorf("invalid QPUSH payload")
	}

	req.Value = payload[pos : pos+int(valueLen)]

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

// EncodeSPublishRequest encodes a SPUBLISH request
func EncodeSPublishRequest(topic string, partition int, key string, value []byte) []byte {
	topicLen := len(topic)
	keyLen := len(key)
	valueLen := len(value)

	// Format: [1:opcode][4:payloadLen][2:topicLen][topic][4:partition][2:keyLen][key][4:valueLen][value]
	totalSize := 1 + 4 + 2 + topicLen + 4 + 2 + keyLen + 4 + valueLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpSPublish
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(topicLen))
	pos += 2
	copy(buf[pos:], topic)
	pos += topicLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(partition))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
	pos += 2
	copy(buf[pos:], key)
	pos += keyLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(valueLen))
	pos += 4
	copy(buf[pos:], value)

	return buf
}

// EncodeSConsumeRequest encodes a SCONSUME request
func EncodeSConsumeRequest(topic, group, consumer string, count int) []byte {
	topicLen := len(topic)
	groupLen := len(group)
	consumerLen := len(consumer)

	// Format: [1:opcode][4:payloadLen][2:topicLen][topic][2:groupLen][group][2:consumerLen][consumer][4:count]
	totalSize := 1 + 4 + 2 + topicLen + 2 + groupLen + 2 + consumerLen + 4
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpSConsume
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(topicLen))
	pos += 2
	copy(buf[pos:], topic)
	pos += topicLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(groupLen))
	pos += 2
	copy(buf[pos:], group)
	pos += groupLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(consumerLen))
	pos += 2
	copy(buf[pos:], consumer)
	pos += consumerLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(count))

	return buf
}

// EncodeSCommitRequest encodes a SCOMMIT request
func EncodeSCommitRequest(topic, group string, partition int, offset uint64) []byte {
	topicLen := len(topic)
	groupLen := len(group)

	// Format: [1:opcode][4:payloadLen][2:topicLen][topic][2:groupLen][group][4:partition][8:offset]
	totalSize := 1 + 4 + 2 + topicLen + 2 + groupLen + 4 + 8
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpSCommit
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(topicLen))
	pos += 2
	copy(buf[pos:], topic)
	pos += topicLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(groupLen))
	pos += 2
	copy(buf[pos:], group)
	pos += groupLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(partition))
	pos += 4

	binary.BigEndian.PutUint64(buf[pos:], offset)

	return buf
}

// EncodeSCreateTopicRequest encodes a SCREATETOPIC request
func EncodeSCreateTopicRequest(name string, partitions int, retentionMs int64) []byte {
	nameLen := len(name)

	// Format: [1:opcode][4:payloadLen][2:nameLen][name][4:partitions][8:retentionMs]
	totalSize := 1 + 4 + 2 + nameLen + 4 + 8
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpSCreateTopic
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(nameLen))
	pos += 2
	copy(buf[pos:], name)
	pos += nameLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(partitions))
	pos += 4

	binary.BigEndian.PutUint64(buf[pos:], uint64(retentionMs))

	return buf
}

// EncodeSSubscribeRequest encodes a SSUBSCRIBE request
func EncodeSSubscribeRequest(topic, group, consumer string) []byte {
	topicLen := len(topic)
	groupLen := len(group)
	consumerLen := len(consumer)

	// Format: [1:opcode][4:payloadLen][2:topicLen][topic][2:groupLen][group][2:consumerLen][consumer]
	totalSize := 1 + 4 + 2 + topicLen + 2 + groupLen + 2 + consumerLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpSSubscribe
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(topicLen))
	pos += 2
	copy(buf[pos:], topic)
	pos += topicLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(groupLen))
	pos += 2
	copy(buf[pos:], group)
	pos += groupLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(consumerLen))
	pos += 2
	copy(buf[pos:], consumer)
	pos += consumerLen

	return buf
}

// EncodeSUnsubscribeRequest encodes a SUNSUBSCRIBE request
func EncodeSUnsubscribeRequest(topic, group, consumer string) []byte {
	topicLen := len(topic)
	groupLen := len(group)
	consumerLen := len(consumer)

	// Format: [1:opcode][4:payloadLen][2:topicLen][topic][2:groupLen][group][2:consumerLen][consumer]
	totalSize := 1 + 4 + 2 + topicLen + 2 + groupLen + 2 + consumerLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpSUnsubscribe
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(topicLen))
	pos += 2
	copy(buf[pos:], topic)
	pos += topicLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(groupLen))
	pos += 2
	copy(buf[pos:], group)
	pos += groupLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(consumerLen))
	pos += 2
	copy(buf[pos:], consumer)
	pos += consumerLen

	return buf
}

// EncodeDocInsertRequest encodes a DOCINSERT request
func EncodeDocInsertRequest(collection string, doc []byte) []byte {
	collLen := len(collection)
	docLen := len(doc)

	// Format: [1:opcode][4:payloadLen][2:collLen][collection][4:docLen][doc]
	totalSize := 1 + 4 + 2 + collLen + 4 + docLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpDocInsert
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(collLen))
	pos += 2
	copy(buf[pos:], collection)
	pos += collLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(docLen))
	pos += 4
	copy(buf[pos:], doc)

	return buf
}

// EncodeDocFindRequest encodes a DOCFIND request
func EncodeDocFindRequest(collection string, query []byte) []byte {
	collLen := len(collection)
	queryLen := len(query)

	// Format: [1:opcode][4:payloadLen][2:collLen][collection][4:queryLen][query]
	totalSize := 1 + 4 + 2 + collLen + 4 + queryLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpDocFind
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(collLen))
	pos += 2
	copy(buf[pos:], collection)
	pos += collLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(queryLen))
	pos += 4
	copy(buf[pos:], query)

	return buf
}

// EncodeDocIndexRequest encodes a DOCINDEX request to create an index
func EncodeDocIndexRequest(collection string, field string) []byte {
	collLen := len(collection)
	fieldLen := len(field)

	// Format: [1:opcode][4:payloadLen][2:collLen][collection][2:fieldLen][field]
	totalSize := 1 + 4 + 2 + collLen + 2 + fieldLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpDocIndex
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(collLen))
	pos += 2
	copy(buf[pos:], collection)
	pos += collLen

	binary.BigEndian.PutUint16(buf[pos:], uint16(fieldLen))
	pos += 2
	copy(buf[pos:], field)

	return buf
}

// EncodeDocUpdateRequest encodes a DOCUPDATE request
func EncodeDocUpdateRequest(collection string, updateOpts []byte) []byte {
	collLen := len(collection)
	optsLen := len(updateOpts)

	// Format: [1:opcode][4:payloadLen][2:collLen][collection][4:optsLen][opts]
	totalSize := 1 + 4 + 2 + collLen + 4 + optsLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpDocUpdate
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(collLen))
	pos += 2
	copy(buf[pos:], collection)
	pos += collLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(optsLen))
	pos += 4
	copy(buf[pos:], updateOpts)

	return buf
}

// EncodeDocDeleteRequest encodes a DOCDELETE request
func EncodeDocDeleteRequest(collection string, deleteOpts []byte) []byte {
	collLen := len(collection)
	optsLen := len(deleteOpts)

	// Format: [1:opcode][4:payloadLen][2:collLen][collection][4:optsLen][opts]
	totalSize := 1 + 4 + 2 + collLen + 4 + optsLen
	buf := make([]byte, totalSize)

	pos := 0
	buf[pos] = OpDocDelete
	pos++

	payloadLen := totalSize - 5
	binary.BigEndian.PutUint32(buf[pos:], uint32(payloadLen))
	pos += 4

	binary.BigEndian.PutUint16(buf[pos:], uint16(collLen))
	pos += 2
	copy(buf[pos:], collection)
	pos += collLen

	binary.BigEndian.PutUint32(buf[pos:], uint32(optsLen))
	pos += 4
	copy(buf[pos:], deleteOpts)

	return buf
}

// Decode functions for stream requests

func decodeSPublishRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid SPUBLISH payload")
	}

	req := &Request{OpCode: OpSPublish}
	pos := 0

	// Topic
	topicLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+topicLen {
		return nil, fmt.Errorf("invalid SPUBLISH payload")
	}
	req.Topic = string(payload[pos : pos+topicLen])
	pos += topicLen

	// Partition
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid SPUBLISH payload")
	}
	req.Partition = int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4

	// Key
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SPUBLISH payload")
	}
	keyLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+keyLen {
		return nil, fmt.Errorf("invalid SPUBLISH payload")
	}
	req.Key = string(payload[pos : pos+keyLen])
	pos += keyLen

	// Value
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid SPUBLISH payload")
	}
	valueLen := int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4
	if len(payload) < pos+valueLen {
		return nil, fmt.Errorf("invalid SPUBLISH payload")
	}
	req.Value = payload[pos : pos+valueLen]

	return req, nil
}

func decodeSConsumeRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid SCONSUME payload")
	}

	req := &Request{OpCode: OpSConsume}
	pos := 0

	// Topic
	topicLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+topicLen {
		return nil, fmt.Errorf("invalid SCONSUME payload")
	}
	req.Topic = string(payload[pos : pos+topicLen])
	pos += topicLen

	// Group
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SCONSUME payload")
	}
	groupLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+groupLen {
		return nil, fmt.Errorf("invalid SCONSUME payload")
	}
	req.Group = string(payload[pos : pos+groupLen])
	pos += groupLen

	// Consumer
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SCONSUME payload")
	}
	consumerLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+consumerLen {
		return nil, fmt.Errorf("invalid SCONSUME payload")
	}
	req.Consumer = string(payload[pos : pos+consumerLen])
	pos += consumerLen

	// Count
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid SCONSUME payload")
	}
	req.Count = int(binary.BigEndian.Uint32(payload[pos:]))

	return req, nil
}

func decodeSCommitRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid SCOMMIT payload")
	}

	req := &Request{OpCode: OpSCommit}
	pos := 0

	// Topic
	topicLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+topicLen {
		return nil, fmt.Errorf("invalid SCOMMIT payload")
	}
	req.Topic = string(payload[pos : pos+topicLen])
	pos += topicLen

	// Group
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SCOMMIT payload")
	}
	groupLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+groupLen {
		return nil, fmt.Errorf("invalid SCOMMIT payload")
	}
	req.Group = string(payload[pos : pos+groupLen])
	pos += groupLen

	// Partition
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid SCOMMIT payload")
	}
	req.Partition = int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4

	// Offset
	if len(payload) < pos+8 {
		return nil, fmt.Errorf("invalid SCOMMIT payload")
	}
	req.RetentionMs = int64(binary.BigEndian.Uint64(payload[pos:])) // Reusing RetentionMs for Offset

	return req, nil
}

func decodeSCreateTopicRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid SCREATETOPIC payload")
	}

	req := &Request{OpCode: OpSCreateTopic}
	pos := 0

	// Name
	nameLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+nameLen {
		return nil, fmt.Errorf("invalid SCREATETOPIC payload")
	}
	req.Topic = string(payload[pos : pos+nameLen])
	pos += nameLen

	// Partitions
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid SCREATETOPIC payload")
	}
	req.Partition = int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4

	// RetentionMs
	if len(payload) < pos+8 {
		return nil, fmt.Errorf("invalid SCREATETOPIC payload")
	}
	req.RetentionMs = int64(binary.BigEndian.Uint64(payload[pos:]))

	return req, nil
}

func decodeSSubscribeRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid SSUBSCRIBE payload")
	}

	req := &Request{OpCode: OpSSubscribe}
	pos := 0

	// Topic
	topicLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+topicLen {
		return nil, fmt.Errorf("invalid SSUBSCRIBE payload")
	}
	req.Topic = string(payload[pos : pos+topicLen])
	pos += topicLen

	// Group
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SSUBSCRIBE payload")
	}
	groupLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+groupLen {
		return nil, fmt.Errorf("invalid SSUBSCRIBE payload")
	}
	req.Group = string(payload[pos : pos+groupLen])
	pos += groupLen

	// Consumer
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SSUBSCRIBE payload")
	}
	consumerLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+consumerLen {
		return nil, fmt.Errorf("invalid SSUBSCRIBE payload")
	}
	req.Consumer = string(payload[pos : pos+consumerLen])

	return req, nil
}

func decodeSUnsubscribeRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid SUNSUBSCRIBE payload")
	}

	req := &Request{OpCode: OpSUnsubscribe}
	pos := 0

	// Topic
	topicLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+topicLen {
		return nil, fmt.Errorf("invalid SUNSUBSCRIBE payload")
	}
	req.Topic = string(payload[pos : pos+topicLen])
	pos += topicLen

	// Group
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SUNSUBSCRIBE payload")
	}
	groupLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+groupLen {
		return nil, fmt.Errorf("invalid SUNSUBSCRIBE payload")
	}
	req.Group = string(payload[pos : pos+groupLen])
	pos += groupLen

	// Consumer
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid SUNSUBSCRIBE payload")
	}
	consumerLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+consumerLen {
		return nil, fmt.Errorf("invalid SUNSUBSCRIBE payload")
	}
	req.Consumer = string(payload[pos : pos+consumerLen])

	return req, nil
}

// Document store decode functions

func decodeDocInsertRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid DOCINSERT payload")
	}

	req := &Request{OpCode: OpDocInsert}
	pos := 0

	// Collection
	collLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+collLen {
		return nil, fmt.Errorf("invalid DOCINSERT payload")
	}
	req.Collection = string(payload[pos : pos+collLen])
	pos += collLen

	// Doc (stored in Value)
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid DOCINSERT payload")
	}
	docLen := int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4
	if len(payload) < pos+docLen {
		return nil, fmt.Errorf("invalid DOCINSERT payload")
	}
	req.Value = payload[pos : pos+docLen]

	return req, nil
}

func decodeDocFindRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid DOCFIND payload")
	}

	req := &Request{OpCode: OpDocFind}
	pos := 0

	// Collection
	collLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+collLen {
		return nil, fmt.Errorf("invalid DOCFIND payload")
	}
	req.Collection = string(payload[pos : pos+collLen])
	pos += collLen

	// Query (stored in Value)
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid DOCFIND payload")
	}
	queryLen := int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4
	if len(payload) < pos+queryLen {
		return nil, fmt.Errorf("invalid DOCFIND payload")
	}
	req.Value = payload[pos : pos+queryLen]

	return req, nil
}

func decodeDocUpdateRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid DOCUPDATE payload")
	}

	req := &Request{OpCode: OpDocUpdate}
	pos := 0

	// Collection
	collLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+collLen {
		return nil, fmt.Errorf("invalid DOCUPDATE payload")
	}
	req.Collection = string(payload[pos : pos+collLen])
	pos += collLen

	// Update Options (stored in Value)
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid DOCUPDATE payload")
	}
	optsLen := int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4
	if len(payload) < pos+optsLen {
		return nil, fmt.Errorf("invalid DOCUPDATE payload")
	}
	req.Value = payload[pos : pos+optsLen]

	return req, nil
}

func decodeDocDeleteRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid DOCDELETE payload")
	}

	req := &Request{OpCode: OpDocDelete}
	pos := 0

	// Collection
	collLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+collLen {
		return nil, fmt.Errorf("invalid DOCDELETE payload")
	}
	req.Collection = string(payload[pos : pos+collLen])
	pos += collLen

	// Query (stored in Value)
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid DOCDELETE payload")
	}
	queryLen := int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4
	if len(payload) < pos+queryLen {
		return nil, fmt.Errorf("invalid DOCDELETE payload")
	}
	req.Value = payload[pos : pos+queryLen]

	return req, nil
}

func decodeDocIndexRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid DOCINDEX payload")
	}

	req := &Request{OpCode: OpDocIndex}
	pos := 0

	// Collection
	collLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+collLen {
		return nil, fmt.Errorf("invalid DOCINDEX payload")
	}
	req.Collection = string(payload[pos : pos+collLen])
	pos += collLen

	// Field (stored in Key)
	if len(payload) < pos+2 {
		return nil, fmt.Errorf("invalid DOCINDEX payload")
	}
	fieldLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+fieldLen {
		return nil, fmt.Errorf("invalid DOCINDEX payload")
	}
	req.Key = string(payload[pos : pos+fieldLen])

	return req, nil
}

func decodeSPublishBatchRequest(payload []byte) (*Request, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
	}

	req := &Request{OpCode: OpSPublishBatch}
	pos := 0

	// Topic
	topicLen := int(binary.BigEndian.Uint16(payload[pos:]))
	pos += 2
	if len(payload) < pos+topicLen {
		return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
	}
	req.Topic = string(payload[pos : pos+topicLen])
	pos += topicLen

	// Count (number of messages)
	if len(payload) < pos+4 {
		return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
	}
	count := int(binary.BigEndian.Uint32(payload[pos:]))
	pos += 4

	if count > MaxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", count, MaxBatchSize)
	}

	req.BatchMessages = make([]*BatchMessage, count)

	for i := 0; i < count; i++ {
		msg := &BatchMessage{}

		// Partition
		if len(payload) < pos+4 {
			return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
		}
		msg.Partition = int(binary.BigEndian.Uint32(payload[pos:]))
		pos += 4

		// Key
		if len(payload) < pos+2 {
			return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
		}
		keyLen := int(binary.BigEndian.Uint16(payload[pos:]))
		pos += 2
		if len(payload) < pos+keyLen {
			return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
		}
		msg.Key = string(payload[pos : pos+keyLen])
		pos += keyLen

		// Value
		if len(payload) < pos+4 {
			return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
		}
		valLen := int(binary.BigEndian.Uint32(payload[pos:]))
		pos += 4
		if len(payload) < pos+valLen {
			return nil, fmt.Errorf("invalid SPUBLISHBATCH payload")
		}
		msg.Value = make([]byte, valLen)
		copy(msg.Value, payload[pos:pos+valLen])
		pos += valLen

		req.BatchMessages[i] = msg
	}

	return req, nil
}

func EncodeSPublishBatchRequest(topic string, messages []*BatchMessage) []byte {
	// Calculate size
	// OpCode(1) + TopicLen(2) + Topic + Count(4) + (Partition(4) + KeyLen(2) + Key + ValueLen(4) + Value)...
	size := 1 + 2 + len(topic) + 4
	for _, msg := range messages {
		size += 4 + 2 + len(msg.Key) + 4 + len(msg.Value)
	}

	buf := make([]byte, size)
	pos := 0

	buf[pos] = OpSPublishBatch
	pos++

	binary.BigEndian.PutUint16(buf[pos:], uint16(len(topic)))
	pos += 2
	copy(buf[pos:], topic)
	pos += len(topic)

	binary.BigEndian.PutUint32(buf[pos:], uint32(len(messages)))
	pos += 4

	for _, msg := range messages {
		binary.BigEndian.PutUint32(buf[pos:], uint32(msg.Partition))
		pos += 4

		binary.BigEndian.PutUint16(buf[pos:], uint16(len(msg.Key)))
		pos += 2
		copy(buf[pos:], msg.Key)
		pos += len(msg.Key)

		binary.BigEndian.PutUint32(buf[pos:], uint32(len(msg.Value)))
		pos += 4
		copy(buf[pos:], msg.Value)
		pos += len(msg.Value)
	}

	return buf
}
