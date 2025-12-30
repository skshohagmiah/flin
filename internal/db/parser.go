package db

import (
	"fmt"
	"strings"
)

// ParseSchema parses a FQL (Flin Query Language) schema definition string
// and returns a list of Schema objects.
func ParseSchema(fql string) ([]Schema, error) {
	var schemas []Schema

	// Normalize line endings and split by semicolon
	statements := strings.Split(fql, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		// Parse CREATE TABLE statement
		if strings.HasPrefix(strings.ToUpper(stmt), "CREATE TABLE") {
			schema, err := parseCreateTable(stmt)
			if err != nil {
				return nil, err
			}
			schemas = append(schemas, schema)
		}
	}

	return schemas, nil
}

func parseCreateTable(stmt string) (Schema, error) {
	var schema Schema
	schema.Columns = make(map[string]FieldType)

	// Extract table name
	// Format: CREATE TABLE tableName ( ... )
	parts := strings.SplitN(stmt, "(", 2)
	if len(parts) != 2 {
		return schema, fmt.Errorf("invalid CREATE TABLE statement: missing '('")
	}

	tableDef := strings.TrimSpace(parts[0])
	// Handle case-insensitivity by taking the original casing from the definition if needed,
	// but usually table names are normalized. Let's just take the last word.
	words := strings.Fields(tableDef)
	if len(words) >= 3 {
		schema.TableName = words[2] // CREATE TABLE <name>
	} else {
		return schema, fmt.Errorf("invalid table name definition")
	}

	// Parse body
	body := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")

	// Helper to split by comma respecting parentheses
	lines := splitByCommaWithParens(body)

	// Track if we found primary key definition
	foundPK := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		if strings.HasPrefix(upperLine, "PRIMARY KEY") {
			foundPK = true
			if err := parsePrimaryKey(line, &schema); err != nil {
				return schema, err
			}
			continue
		}

		// Column definition: name type
		colParts := strings.Fields(line)
		if len(colParts) < 2 {
			continue
		}
		colName := colParts[0]
		colType := strings.ToLower(colParts[1])

		// Map FQL/CQL types to internal types
		switch colType {
		case "uuid", "text", "varchar":
			schema.Columns[colName] = TypeString
		case "int", "bigint", "integer":
			schema.Columns[colName] = TypeInt
		case "double", "float":
			schema.Columns[colName] = TypeFloat
		case "boolean":
			schema.Columns[colName] = TypeBool
		case "timestamp":
			// We store timestamps as milliseconds int64
			schema.Columns[colName] = TypeInt
		default:
			schema.Columns[colName] = TypeString // fallback
		}
	}

	if !foundPK {
		return schema, fmt.Errorf("table %s missing PRIMARY KEY definition", schema.TableName)
	}

	return schema, nil
}

func parsePrimaryKey(line string, schema *Schema) error {
	// Format: PRIMARY KEY ( (part1, part2), clust1, clust2 )
	//     or: PRIMARY KEY ( part1, clust1 )
	//     or: PRIMARY KEY ( part1 )

	// Extract content inside outermost parens
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start == -1 || end == -1 {
		return fmt.Errorf("invalid PRIMARY KEY syntax")
	}

	content := strings.TrimSpace(line[start+1 : end])

	// Check for nested Parens indicating composite partition key
	if strings.HasPrefix(content, "(") {
		// Composite Partition Key: ((p1, p2), c1)
		// Find end of partition group
		pEnd := strings.Index(content, ")")
		if pEnd == -1 {
			return fmt.Errorf("invalid composite partition key syntax")
		}

		pContent := content[1:pEnd]
		pKeys := splitByCommaWithParens(pContent)
		for _, k := range pKeys {
			schema.PartitionKeys = append(schema.PartitionKeys, strings.TrimSpace(k))
		}

		// Rest are clustering keys
		if pEnd+1 < len(content) {
			rest := content[pEnd+1:]
			rest = strings.TrimPrefix(strings.TrimSpace(rest), ",")
			if len(rest) > 0 {
				cKeys := splitByCommaWithParens(rest)
				for _, k := range cKeys {
					schema.ClusteringKeys = append(schema.ClusteringKeys, strings.TrimSpace(k))
				}
			}
		}

	} else {
		// Simple Partition Key: (p1, c1, c2) -> First is partition, rest are clustering
		keys := splitByCommaWithParens(content)
		if len(keys) == 0 {
			return fmt.Errorf("empty PRIMARY KEY")
		}

		schema.PartitionKeys = append(schema.PartitionKeys, strings.TrimSpace(keys[0]))

		if len(keys) > 1 {
			for _, k := range keys[1:] {
				schema.ClusteringKeys = append(schema.ClusteringKeys, strings.TrimSpace(k))
			}
		}
	}

	return nil
}

// splitByCommaWithParens splits a string by comma, respecting parentheses checks
func splitByCommaWithParens(s string) []string {
	var parts []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range s {
		if char == '(' {
			parenDepth++
		} else if char == ')' {
			parenDepth--
		}

		if char == ',' && parenDepth == 0 {
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteRune(char)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}
	return parts
}
