package compiler

import (
	"testing"

	"github.com/sqlc-dev/sqlc/internal/sql/ast"
)

func TestExcludeColumnsFilter_ShouldExclude(t *testing.T) {
	tests := []struct {
		name           string
		excludeColumns []string
		tableName      *ast.TableName
		columnName     string
		want           bool
	}{
		{
			name:           "table.column format matches",
			excludeColumns: []string{"users.password"},
			tableName:      &ast.TableName{Name: "users"},
			columnName:     "password",
			want:           true,
		},
		{
			name:           "table.column format does not match different column",
			excludeColumns: []string{"users.password"},
			tableName:      &ast.TableName{Name: "users"},
			columnName:     "name",
			want:           false,
		},
		{
			name:           "table.column format does not match different table",
			excludeColumns: []string{"users.password"},
			tableName:      &ast.TableName{Name: "admins"},
			columnName:     "password",
			want:           false,
		},
		{
			name:           "schema.table.column format matches",
			excludeColumns: []string{"public.users.password"},
			tableName:      &ast.TableName{Schema: "public", Name: "users"},
			columnName:     "password",
			want:           true,
		},
		{
			name:           "schema.table.column format does not match different schema",
			excludeColumns: []string{"public.users.password"},
			tableName:      &ast.TableName{Schema: "private", Name: "users"},
			columnName:     "password",
			want:           false,
		},
		{
			name:           "case insensitive match",
			excludeColumns: []string{"Users.Password"},
			tableName:      &ast.TableName{Name: "users"},
			columnName:     "password",
			want:           true,
		},
		{
			name:           "case insensitive match with schema",
			excludeColumns: []string{"Public.Users.Password"},
			tableName:      &ast.TableName{Schema: "public", Name: "users"},
			columnName:     "password",
			want:           true,
		},
		{
			name:           "empty filter excludes nothing",
			excludeColumns: []string{},
			tableName:      &ast.TableName{Name: "users"},
			columnName:     "password",
			want:           false,
		},
		{
			name:           "nil table name excludes nothing",
			excludeColumns: []string{"users.password"},
			tableName:      nil,
			columnName:     "password",
			want:           false,
		},
		{
			name:           "multiple exclusions",
			excludeColumns: []string{"users.password", "users.ssn", "accounts.secret"},
			tableName:      &ast.TableName{Name: "users"},
			columnName:     "ssn",
			want:           true,
		},
		{
			name:           "table.column matches even when schema present in tableName",
			excludeColumns: []string{"users.password"},
			tableName:      &ast.TableName{Schema: "public", Name: "users"},
			columnName:     "password",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewExcludeColumnsFilter(tt.excludeColumns)
			got := filter.ShouldExclude(tt.tableName, tt.columnName)
			if got != tt.want {
				t.Errorf("ShouldExclude() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcludeColumnsFilter_NilFilter(t *testing.T) {
	var filter *ExcludeColumnsFilter
	got := filter.ShouldExclude(&ast.TableName{Name: "users"}, "password")
	if got != false {
		t.Errorf("ShouldExclude() on nil filter = %v, want false", got)
	}
}
