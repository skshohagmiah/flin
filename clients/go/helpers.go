package flin

import (
	"encoding/binary"
	"errors"

	"github.com/skshohagmiah/flin/internal/net"
	protocol "github.com/skshohagmiah/flin/internal/net"
)

// Helper functions for reading responses

func readOKResponse(conn *net.Connection) error {
	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return err
	}

	if status != protocol.StatusOK {
		if payloadLen > 0 {
			payload, err := conn.Read(int(payloadLen))
			if err != nil {
				return err
			}
			return errors.New(string(payload))
		}
		return errors.New("operation failed")
	}

	// Consume payload if any
	if payloadLen > 0 {
		_, err = conn.Read(int(payloadLen))
	}

	return err
}

func readValueResponse(conn *net.Connection) ([]byte, error) {
	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return nil, err
	}

	if status == protocol.StatusNotFound {
		return nil, errors.New("key not found")
	}

	if status != protocol.StatusOK {
		if payloadLen > 0 {
			payload, err := conn.Read(int(payloadLen))
			if err != nil {
				return nil, err
			}
			return nil, errors.New(string(payload))
		}
		return nil, errors.New("operation failed")
	}

	if payloadLen == 0 {
		return []byte{}, nil
	}

	return conn.Read(int(payloadLen))
}

func readMultiValueResponse(conn *net.Connection) ([][]byte, error) {
	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return nil, err
	}

	if status != protocol.StatusMultiValue && status != protocol.StatusOK {
		if payloadLen > 0 {
			payload, err := conn.Read(int(payloadLen))
			if err != nil {
				return nil, err
			}
			return nil, errors.New(string(payload))
		}
		return nil, errors.New("operation failed")
	}

	if payloadLen == 0 {
		return [][]byte{}, nil
	}

	payload, err := conn.Read(int(payloadLen))
	if err != nil {
		return nil, err
	}

	// Parse multi-value response
	if len(payload) < 2 {
		return nil, errors.New("invalid multi-value response")
	}

	count := binary.BigEndian.Uint16(payload[0:2])
	values := make([][]byte, 0, count)
	pos := 2

	for i := 0; i < int(count); i++ {
		if len(payload) < pos+4 {
			return nil, errors.New("invalid multi-value response")
		}

		valueLen := binary.BigEndian.Uint32(payload[pos:])
		pos += 4

		if len(payload) < pos+int(valueLen) {
			return nil, errors.New("invalid multi-value response")
		}

		value := make([]byte, valueLen)
		copy(value, payload[pos:pos+int(valueLen)])
		pos += int(valueLen)

		values = append(values, value)
	}

	return values, nil
}
