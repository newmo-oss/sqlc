package compiler

import (
	"strings"

	"github.com/sqlc-dev/sqlc/internal/sql/ast"
)

// ExcludeColumnsFilter filters columns that should be excluded from star expansion.
type ExcludeColumnsFilter struct {
	// excludeMap holds normalized (lowercase) column identifiers.
	// Keys are in "table.column" or "schema.table.column" format.
	excludeMap map[string]struct{}
}

// NewExcludeColumnsFilter creates a new ExcludeColumnsFilter from a list of column identifiers.
// Each identifier should be in "table.column" or "schema.table.column" format.
func NewExcludeColumnsFilter(excludeColumns []string) *ExcludeColumnsFilter {
	m := make(map[string]struct{})
	for _, col := range excludeColumns {
		m[strings.ToLower(col)] = struct{}{}
	}
	return &ExcludeColumnsFilter{excludeMap: m}
}

// ShouldExclude returns true if the specified column should be excluded from star expansion.
// tableName should be the actual table name (not an alias).
func (f *ExcludeColumnsFilter) ShouldExclude(tableName *ast.TableName, columnName string) bool {
	if f == nil || len(f.excludeMap) == 0 || tableName == nil {
		return false
	}

	// Check "table.column" format
	key := strings.ToLower(tableName.Name + "." + columnName)
	if _, ok := f.excludeMap[key]; ok {
		return true
	}

	// Check "schema.table.column" format
	if tableName.Schema != "" {
		key = strings.ToLower(tableName.Schema + "." + tableName.Name + "." + columnName)
		if _, ok := f.excludeMap[key]; ok {
			return true
		}
	}

	return false
}
