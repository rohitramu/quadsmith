package schemacheck

import (
	"fmt"
	"regexp"
	"strings"
)

// Severity indicates how critical a detected change is.
type Severity string

const (
	SeverityBreaking Severity = "BREAKING"
	SeverityWarning  Severity = "WARNING"
)

// ChangeType identifies the specific type of schema difference.
type ChangeType string

const (
	ChangeTableDropped       ChangeType = "TABLE_DROPPED"
	ChangeTableAdded         ChangeType = "TABLE_ADDED"
	ChangeColumnDropped      ChangeType = "COLUMN_DROPPED"
	ChangeColumnAdded        ChangeType = "COLUMN_ADDED"
	ChangeTypeChanged        ChangeType = "COLUMN_TYPE_CHANGED"
	ChangeColumnMadeNotNull  ChangeType = "COLUMN_MADE_NOT_NULL"
	ChangeNewColumnNotNull   ChangeType = "NEW_COLUMN_NOT_NULL"
	ChangePrimaryKeyModified ChangeType = "PRIMARY_KEY_MODIFIED"
	ChangeUniqueAdded        ChangeType = "UNIQUE_CONSTRAINT_ADDED"
	ChangeForeignKeyDropped  ChangeType = "FOREIGN_KEY_DROPPED"
)

// BreakingChange describes a backwards-incompatible database schema alteration.
type BreakingChange struct {
	Type        ChangeType
	Table       string
	Column      string
	Description string
	Severity    Severity
}

func (b BreakingChange) String() string {
	if b.Column != "" {
		return fmt.Sprintf("[%s] %s (%s.%s): %s", b.Severity, b.Type, b.Table, b.Column, b.Description)
	}
	return fmt.Sprintf("[%s] %s (%s): %s", b.Severity, b.Type, b.Table, b.Description)
}

// Column represents a table column definition.
type Column struct {
	Name         string
	DataType     string
	IsPrimaryKey bool
	IsUnique     bool
	IsNotNull    bool
	HasDefault   bool
	DefaultValue string
}

// Table represents a parsed table schema.
type Table struct {
	Name        string
	Columns     map[string]*Column
	ColumnOrder []string
	PrimaryKey  string
}

// ForeignKey represents a foreign key constraint.
type ForeignKey struct {
	ConstraintName string
	Table          string
	Column         string
	RefTable       string
	RefColumn      string
}

// Schema represents the complete parsed database schema.
type Schema struct {
	Tables      map[string]*Table
	ForeignKeys map[string]*ForeignKey
	Indexes     map[string]string // indexName -> statement
}

// NewSchema allocates an empty Schema.
func NewSchema() *Schema {
	return &Schema{
		Tables:      make(map[string]*Table),
		ForeignKeys: make(map[string]*ForeignKey),
		Indexes:     make(map[string]string),
	}
}

var (
	createTableRegex  = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_"]+)\s*\(([\s\S]*?)\);`)
	alterTableFkRegex = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+([a-zA-Z0-9_"]+)\s+ADD\s+CONSTRAINT\s+([a-zA-Z0-9_"]+)\s+FOREIGN\s+KEY\s*\(([a-zA-Z0-9_"]+)\)\s+REFERENCES\s+([a-zA-Z0-9_"]+)\s*\(([a-zA-Z0-9_"]+)\);`)
	createIndexRegex  = regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_"]+)\s+ON\s+([a-zA-Z0-9_"]+)\s*\(([a-zA-Z0-9_"]+)\);`)
)

func cleanIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return s
}

// ParseSQL parses PostgreSQL DDL into a structured Schema representation.
func ParseSQL(sqlContent string) (*Schema, error) {
	schema := NewSchema()

	// 1. Parse CREATE TABLE statements
	tableMatches := createTableRegex.FindAllStringSubmatch(sqlContent, -1)
	for _, match := range tableMatches {
		tableName := cleanIdent(match[1])
		body := match[2]

		table := &Table{
			Name:    tableName,
			Columns: make(map[string]*Column),
		}

		// Split definitions by comma (handling nested types if needed)
		lines := splitDefinitions(body)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Check for PRIMARY KEY (col)
			if strings.HasPrefix(strings.ToUpper(line), "PRIMARY KEY") {
				pkRegex := regexp.MustCompile(`(?i)PRIMARY\s+KEY\s*\(([a-zA-Z0-9_"]+)\)`)
				if pkMatch := pkRegex.FindStringSubmatch(line); pkMatch != nil {
					pkCol := cleanIdent(pkMatch[1])
					table.PrimaryKey = pkCol
					if col, ok := table.Columns[pkCol]; ok {
						col.IsPrimaryKey = true
					}
				}
				continue
			}

			// Parse column definition: <name> <type> [UNIQUE] [NOT NULL] [DEFAULT ...]
			col := parseColumnDef(line)
			if col != nil {
				table.Columns[col.Name] = col
				table.ColumnOrder = append(table.ColumnOrder, col.Name)
				if col.IsPrimaryKey {
					table.PrimaryKey = col.Name
				}
			}
		}

		schema.Tables[tableName] = table
	}

	// 2. Parse Foreign Keys
	fkMatches := alterTableFkRegex.FindAllStringSubmatch(sqlContent, -1)
	for _, match := range fkMatches {
		tbl := cleanIdent(match[1])
		cName := cleanIdent(match[2])
		col := cleanIdent(match[3])
		refTbl := cleanIdent(match[4])
		refCol := cleanIdent(match[5])

		fk := &ForeignKey{
			ConstraintName: cName,
			Table:          tbl,
			Column:         col,
			RefTable:       refTbl,
			RefColumn:      refCol,
		}
		schema.ForeignKeys[cName] = fk
	}

	// 3. Parse Indexes
	idxMatches := createIndexRegex.FindAllStringSubmatch(sqlContent, -1)
	for _, match := range idxMatches {
		idxName := cleanIdent(match[1])
		schema.Indexes[idxName] = match[0]
	}

	return schema, nil
}

func splitDefinitions(body string) []string {
	var defs []string
	var cur strings.Builder
	parenDepth := 0

	for _, ch := range body {
		switch ch {
		case '(':
			parenDepth++
			cur.WriteRune(ch)
		case ')':
			parenDepth--
			cur.WriteRune(ch)
		case ',':
			if parenDepth == 0 {
				defs = append(defs, cur.String())
				cur.Reset()
			} else {
				cur.WriteRune(ch)
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		defs = append(defs, cur.String())
	}
	return defs
}

func parseColumnDef(line string) *Column {
	tokens := strings.Fields(line)
	if len(tokens) < 2 {
		return nil
	}

	name := cleanIdent(tokens[0])
	upperLine := strings.ToUpper(line)

	col := &Column{
		Name: name,
	}

	// Flags
	if strings.Contains(upperLine, "PRIMARY KEY") {
		col.IsPrimaryKey = true
	}
	if strings.Contains(upperLine, "NOT NULL") {
		col.IsNotNull = true
	}
	if strings.Contains(upperLine, "UNIQUE") {
		col.IsUnique = true
	}
	if strings.Contains(upperLine, "DEFAULT") {
		col.HasDefault = true
	}

	// Determine data type
	rawType := tokens[1]
	// If type is something like "CHARACTER VARYING", tokens might have spaces
	typeTokens := make([]string, 0, len(tokens)-1)
	for i := 1; i < len(tokens); i++ {
		tUpper := strings.ToUpper(tokens[i])
		if tUpper == "NOT" || tUpper == "NULL" || tUpper == "UNIQUE" || tUpper == "PRIMARY" || tUpper == "KEY" || tUpper == "DEFAULT" {
			break
		}
		typeTokens = append(typeTokens, tokens[i])
	}
	if len(typeTokens) > 0 {
		rawType = strings.Join(typeTokens, " ")
	}

	col.DataType = normalizeDataType(rawType)
	return col
}

func normalizeDataType(dt string) string {
	dt = strings.ToUpper(strings.TrimSpace(dt))
	switch dt {
	case "INT", "INTEGER", "INT4":
		return "INTEGER"
	case "BIGINT", "INT8":
		return "BIGINT"
	case "SMALLINT", "INT2":
		return "SMALLINT"
	case "DECIMAL", "NUMERIC":
		return "DECIMAL"
	case "REAL", "FLOAT4":
		return "REAL"
	case "DOUBLE PRECISION", "FLOAT8":
		return "DOUBLE PRECISION"
	case "BOOLEAN", "BOOL":
		return "BOOLEAN"
	case "VARCHAR", "CHARACTER VARYING", "TEXT":
		return "TEXT"
	case "UUID":
		return "UUID"
	case "JSON", "JSONB":
		return "JSONB"
	case "UUID[]":
		return "UUID[]"
	case "TEXT[]":
		return "TEXT[]"
	default:
		return dt
	}
}

// Compare compares baseline and current schemas and returns all detected breaking changes.
func Compare(baseline, current *Schema) []BreakingChange {
	var breaking []BreakingChange

	if baseline == nil || current == nil {
		return breaking
	}

	// 1. Check for dropped tables
	for tableName, baseTable := range baseline.Tables {
		currTable, exists := current.Tables[tableName]
		if !exists {
			breaking = append(breaking, BreakingChange{
				Type:        ChangeTableDropped,
				Table:       tableName,
				Description: fmt.Sprintf("Table %q was dropped from the schema.", tableName),
				Severity:    SeverityBreaking,
			})
			continue
		}

		// Check primary key changes
		if baseTable.PrimaryKey != "" && currTable.PrimaryKey != "" && baseTable.PrimaryKey != currTable.PrimaryKey {
			breaking = append(breaking, BreakingChange{
				Type:        ChangePrimaryKeyModified,
				Table:       tableName,
				Column:      currTable.PrimaryKey,
				Description: fmt.Sprintf("Primary key for table %q was changed from %q to %q.", tableName, baseTable.PrimaryKey, currTable.PrimaryKey),
				Severity:    SeverityBreaking,
			})
		}

		// 2. Check for dropped columns or column modifications
		for colName, baseCol := range baseTable.Columns {
			currCol, colExists := currTable.Columns[colName]
			if !colExists {
				breaking = append(breaking, BreakingChange{
					Type:        ChangeColumnDropped,
					Table:       tableName,
					Column:      colName,
					Description: fmt.Sprintf("Column %q was dropped from table %q.", colName, tableName),
					Severity:    SeverityBreaking,
				})
				continue
			}

			// Incompatible data type change
			if baseCol.DataType != currCol.DataType {
				breaking = append(breaking, BreakingChange{
					Type:        ChangeTypeChanged,
					Table:       tableName,
					Column:      colName,
					Description: fmt.Sprintf("Column %q in table %q changed type from %s to %s.", colName, tableName, baseCol.DataType, currCol.DataType),
					Severity:    SeverityBreaking,
				})
			}

			// Existing column changed from nullable to NOT NULL
			if !baseCol.IsNotNull && currCol.IsNotNull && !currCol.HasDefault {
				breaking = append(breaking, BreakingChange{
					Type:        ChangeColumnMadeNotNull,
					Table:       tableName,
					Column:      colName,
					Description: fmt.Sprintf("Existing column %q in table %q was changed from nullable to NOT NULL without a DEFAULT value.", colName, tableName),
					Severity:    SeverityBreaking,
				})
			}

			// UNIQUE constraint added to existing column
			if !baseCol.IsUnique && currCol.IsUnique {
				breaking = append(breaking, BreakingChange{
					Type:        ChangeUniqueAdded,
					Table:       tableName,
					Column:      colName,
					Description: fmt.Sprintf("UNIQUE constraint added to existing column %q in table %q.", colName, tableName),
					Severity:    SeverityBreaking,
				})
			}
		}

		// 3. Check for newly added columns that are NOT NULL without a DEFAULT
		for colName, currCol := range currTable.Columns {
			if _, existed := baseTable.Columns[colName]; !existed {
				if currCol.IsNotNull && !currCol.HasDefault {
					breaking = append(breaking, BreakingChange{
						Type:        ChangeNewColumnNotNull,
						Table:       tableName,
						Column:      colName,
						Description: fmt.Sprintf("Newly added column %q in table %q is NOT NULL without a DEFAULT value.", colName, tableName),
						Severity:    SeverityBreaking,
					})
				}
			}
		}
	}

	// 4. Check for dropped foreign keys
	for fkName, baseFk := range baseline.ForeignKeys {
		if _, exists := current.ForeignKeys[fkName]; !exists {
			// Check if referencing table was dropped entirely (already flagged)
			if _, tblExists := current.Tables[baseFk.Table]; tblExists {
				breaking = append(breaking, BreakingChange{
					Type:        ChangeForeignKeyDropped,
					Table:       baseFk.Table,
					Column:      baseFk.Column,
					Description: fmt.Sprintf("Foreign key constraint %q on %s(%s) referencing %s(%s) was dropped.", fkName, baseFk.Table, baseFk.Column, baseFk.RefTable, baseFk.RefColumn),
					Severity:    SeverityBreaking,
				})
			}
		}
	}

	return breaking
}
