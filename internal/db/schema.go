package db

import (
	"fmt"
	"strings"
)

// FieldType represents the data type of a field
type FieldType string

const (
	TypeString FieldType = "string"
	TypeInt    FieldType = "int"
	TypeFloat  FieldType = "float"
	TypeBool   FieldType = "bool"
	TypeMap    FieldType = "map"
	TypeArray  FieldType = "array"
	TypeBytes  FieldType = "bytes"
	TypeAny    FieldType = "any"
)

// Schema defines the structure of a document collection
type Schema struct {
	TableName      string               `json:"table_name"`
	PartitionKeys  []string             `json:"partition_keys"`  // Support composite partition keys
	ClusteringKeys []string             `json:"clustering_keys"` // Optional sorting keys
	Columns        map[string]FieldType `json:"columns"`
}

// Validate checks if a document conforms to the schema
func (s *Schema) Validate(doc map[string]interface{}) error {
	// Check required partition keys
	for _, pk := range s.PartitionKeys {
		if _, ok := doc[pk]; !ok {
			return fmt.Errorf("missing partition key: %s", pk)
		}
	}

	// Check required clustering keys
	for _, ck := range s.ClusteringKeys {
		if _, ok := doc[ck]; !ok {
			return fmt.Errorf("missing clustering key: %s", ck)
		}
	}

	// Check field types
	for field, expectedType := range s.Columns {
		val, ok := doc[field]
		if !ok {
			continue // Optional fields? For now, yes.
		}

		if err := validateType(val, expectedType); err != nil {
			return fmt.Errorf("field '%s': %w", field, err)
		}
	}

	return nil
}

func validateType(val interface{}, expected FieldType) error {
	if expected == TypeAny {
		return nil
	}

	switch expected {
	case TypeString:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
	case TypeInt:
		// JSON unmarshals numbers as float64 by default
		switch val.(type) {
		case int, int8, int16, int32, int64, float64:
			return nil
		default:
			return fmt.Errorf("expected int, got %T", val)
		}
	case TypeFloat:
		switch val.(type) {
		case float32, float64:
			return nil
		default:
			return fmt.Errorf("expected float, got %T", val)
		}
	case TypeBool:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", val)
		}
	case TypeMap:
		if _, ok := val.(map[string]interface{}); !ok {
			return fmt.Errorf("expected map, got %T", val)
		}
	case TypeArray:
		if _, ok := val.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", val)
		}
	}
	return nil
}

// GenerateKey generates the storage key based on partition and clustering keys
// Format: doc:<table_name>:<partition_hash>:<clustering_values>
// Partition hash is used to distribute data across nodes
// Clustering values are used to sort data within a partition
func (s *Schema) GenerateKey(doc map[string]interface{}) (string, error) {
	// 1. Build Partition Key part
	var pkValues []string
	for _, pk := range s.PartitionKeys {
		val, ok := doc[pk]
		if !ok {
			return "", fmt.Errorf("missing partition key: %s", pk)
		}
		pkValues = append(pkValues, fmt.Sprintf("%v", val))
	}
	partitionString := strings.Join(pkValues, ":")

	// 2. Build Clustering Key part (optional)
	var ckValues []string
	for _, ck := range s.ClusteringKeys {
		val, ok := doc[ck]
		if !ok {
			// If clustering key is missing, maybe we shouldn't fail if we are just generating a partition prefix?
			// But for a full ID, we need it.
			return "", fmt.Errorf("missing clustering key: %s", ck)
		}
		ckValues = append(ckValues, fmt.Sprintf("%v", val))
	}
	clusteringString := strings.Join(ckValues, ":")

	if clusteringString != "" {
		return fmt.Sprintf("doc:%s:%s:%s", s.TableName, partitionString, clusteringString), nil
	}

	return fmt.Sprintf("doc:%s:%s", s.TableName, partitionString), nil
}

// GeneratePartitionPrefix generates the key prefix based ONLY on partition keys
// This is used for queries that filter by partition key but not clustering keys
func (s *Schema) GeneratePartitionPrefix(doc map[string]interface{}) (string, error) {
	var pkValues []string
	for _, pk := range s.PartitionKeys {
		val, ok := doc[pk]
		if !ok {
			return "", fmt.Errorf("missing partition key: %s", pk)
		}
		pkValues = append(pkValues, fmt.Sprintf("%v", val))
	}
	partitionString := strings.Join(pkValues, ":")

	// Return prefix including the trailing colon so it matches strictly within this partition
	// Format: doc:<table_name>:<partition_hash>:
	return fmt.Sprintf("doc:%s:%s:", s.TableName, partitionString), nil
}
