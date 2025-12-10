package cmd

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/sqlc/internal/compiler"
	"github.com/sqlc-dev/sqlc/internal/config"
	"github.com/sqlc-dev/sqlc/internal/config/convert"
	"github.com/sqlc-dev/sqlc/internal/info"
	"github.com/sqlc-dev/sqlc/internal/plugin"
	"github.com/sqlc-dev/sqlc/internal/sql/ast"
	"github.com/sqlc-dev/sqlc/internal/sql/catalog"
)

func pluginSettings(r *compiler.Result, cs config.CombinedSettings) *plugin.Settings {
	return &plugin.Settings{
		Version: cs.Global.Version,
		Engine:  string(cs.Package.Engine),
		Schema:  []string(cs.Package.Schema),
		Queries: []string(cs.Package.Queries),
		Codegen: pluginCodegen(cs, cs.Codegen),
	}
}

func pluginCodegen(cs config.CombinedSettings, s config.Codegen) *plugin.Codegen {
	opts, err := convert.YAMLtoJSON(s.Options)
	if err != nil {
		panic(err)
	}
	cg := &plugin.Codegen{
		Out:     s.Out,
		Plugin:  s.Plugin,
		Options: opts,
	}
	for _, p := range cs.Global.Plugins {
		if p.Name == s.Plugin {
			cg.Env = p.Env
			cg.Process = pluginProcess(p)
			cg.Wasm = pluginWASM(p)
			return cg
		}
	}
	return cg
}

func pluginProcess(p config.Plugin) *plugin.Codegen_Process {
	if p.Process != nil {
		return &plugin.Codegen_Process{
			Cmd: p.Process.Cmd,
		}
	}
	return nil
}

func pluginWASM(p config.Plugin) *plugin.Codegen_WASM {
	if p.WASM != nil {
		return &plugin.Codegen_WASM{
			Url:    p.WASM.URL,
			Sha256: p.WASM.SHA256,
		}
	}
	return nil
}

func pluginCatalog(c *catalog.Catalog) *plugin.Catalog {
	var schemas []*plugin.Schema
	for _, s := range c.Schemas {
		var enums []*plugin.Enum
		var cts []*plugin.CompositeType
		for _, typ := range s.Types {
			switch typ := typ.(type) {
			case *catalog.Enum:
				vals := make([]*plugin.EnumValue, len(typ.Vals))
				for i := range typ.Vals {
					vals[i] = &plugin.EnumValue{
						Value:          typ.EnumVals[i].Val,
						SourceLocation: pluginSourceLocation(typ.EnumVals[i].SourceLocation),
					}
				}

				enums = append(enums, &plugin.Enum{
					Name:           typ.Name,
					Comment:        typ.Comment,
					Vals:           typ.Vals,
					Values:         vals,
					SourceLocation: pluginSourceLocation(typ.SourceLocation),
				})
			case *catalog.CompositeType:
				cts = append(cts, &plugin.CompositeType{
					Name:    typ.Name,
					Comment: typ.Comment,
				})
			}
		}
		var tables []*plugin.Table
		for _, t := range s.Tables {
			var columns []*plugin.Column
			var excludedCols []*plugin.Column
			for _, c := range t.Columns {
				l := -1
				if c.Length != nil {
					l = *c.Length
				}
				col := &plugin.Column{
					Name: c.Name,
					Type: &plugin.Identifier{
						Catalog: c.Type.Catalog,
						Schema:  c.Type.Schema,
						Name:    c.Type.Name,
					},
					Comment:      c.Comment,
					NotNull:      c.IsNotNull,
					Unsigned:     c.IsUnsigned,
					IsGenerated:  c.IsGenerated,
					GenerateExpr: c.GenerateExpr,
					HasDefault:   c.HasDefault,
					DefaultExpr:  c.DefaultExpr,
					IsArray:      c.IsArray,
					ArrayDims:    int32(c.ArrayDims),
					Length:       int32(l),
					Table: &plugin.Identifier{
						Catalog: t.Rel.Catalog,
						Schema:  t.Rel.Schema,
						Name:    t.Rel.Name,
					},
					SourceLocation: pluginSourceLocation(c.SourceLocation),
					TypeMods:       pluginTypeMods(c.Type.Typmods),
				}
				if c.IsExcluded {
					excludedCols = append(excludedCols, col)
				} else {
					columns = append(columns, col)
				}
			}
			var indexes []*plugin.Index
			for _, idx := range t.Indexes {
				var elems []*plugin.IndexElem
				for _, elem := range idx.Elems {
					elems = append(elems, &plugin.IndexElem{
						Name:          elem.Name,
						Ordering:      plugin.SortByDir(elem.Ordering),
						NullsOrdering: plugin.SortByNulls(elem.NullsOrdering),
					})
				}

				indexes = append(indexes, &plugin.Index{
					Name:           idx.Name,
					Elems:          elems,
					IsUnique:       idx.IsUnique,
					IsPrimary:      idx.IsPrimary,
					IsPartial:      idx.IsPartial,
					WhereClause:    idx.WhereClause,
					SourceLocation: pluginSourceLocation(idx.SourceLocation),
				})
			}
			tables = append(tables, &plugin.Table{
				Rel: &plugin.Identifier{
					Catalog: t.Rel.Catalog,
					Schema:  t.Rel.Schema,
					Name:    t.Rel.Name,
				},
				Columns:         columns,
				ExcludedColumns: excludedCols,
				Comment:         t.Comment,
				Indexes:         indexes,
				Constraints:     pluginTableConstraints(t.Constraints),
				SourceLocation:  pluginSourceLocation(t.SourceLocation),
			})
		}
		schemas = append(schemas, &plugin.Schema{
			Comment:        s.Comment,
			Name:           s.Name,
			Tables:         tables,
			Enums:          enums,
			CompositeTypes: cts,
			SourceLocation: pluginSourceLocation(s.SourceLocation),
		})
	}
	return &plugin.Catalog{
		Name:          c.Name,
		DefaultSchema: c.DefaultSchema,
		Comment:       c.Comment,
		Schemas:       schemas,
	}
}

func pluginTableConstraints(constraints *catalog.TableConstraints) *plugin.TableConstraints {
	if constraints == nil {
		return nil
	}

	var fks []*plugin.ForeginKeyConstraint
	for _, c := range constraints.ForeignKeys {
		fks = append(fks, &plugin.ForeginKeyConstraint{
			Rel: &plugin.Identifier{
				Catalog: c.Rel.Catalog,
				Schema:  c.Rel.Schema,
				Name:    c.Rel.Name,
			},
			RelColumnNames: c.RelColumns,
			ColumnNames:    c.Columns,
		})
	}
	return &plugin.TableConstraints{
		ForeignKeys: fks,
	}
}

func pluginSourceLocation(loc *catalog.SourceLocation) *plugin.SourceLocation {
	if loc == nil {
		return nil
	}
	return &plugin.SourceLocation{
		Filename:                loc.Filename,
		StartLine:               int32(loc.StartLine),
		StartColumn:             int32(loc.StartColumn),
		EndLine:                 0,
		EndColumn:               0,
		LeadingDetachedComments: loc.LeadingDetachedComments,
		LeadingComments:         loc.LeadingComments,
		TrailingComments:        loc.TrailingComments,
	}
}

func pluginTypeMods(typmods *ast.List) []string {
	if typmods == nil {
		return nil
	}

	var ss []string
	for _, item := range typmods.Items {
		switch n := item.(type) {
		case *ast.ColumnRef:
			var fields []string
			for _, field := range n.Fields.Items {
				switch v := field.(type) {
				case *ast.String:
					fields = append(fields, v.Str)
				}
			}
			ss = append(ss, strings.Join(fields, "."))
		case *ast.A_Const:
			switch v := n.Val.(type) {
			case *ast.Integer:
				ss = append(ss, fmt.Sprintf("%d", v.Ival))
			case *ast.Float:
				ss = append(ss, v.Str)
			case *ast.Boolean:
				ss = append(ss, fmt.Sprintf("%v", v.Boolval))
			case *ast.String:
				ss = append(ss, fmt.Sprintf("'%s'", v.Str))
			case *ast.BitString:
				ss = append(ss, v.Str)
			}
		}
	}
	return ss
}

func pluginQueries(r *compiler.Result) []*plugin.Query {
	var out []*plugin.Query
	for _, q := range r.Queries {
		var params []*plugin.Parameter
		var columns []*plugin.Column
		var excludedColumns []*plugin.Column
		for _, c := range q.Columns {
			columns = append(columns, pluginQueryColumn(c))
		}
		for _, c := range q.ExcludedColumns {
			excludedColumns = append(excludedColumns, pluginQueryColumn(c))
		}
		for _, p := range q.Params {
			params = append(params, pluginQueryParam(p))
		}
		var iit *plugin.Identifier
		if q.InsertIntoTable != nil {
			iit = &plugin.Identifier{
				Catalog: q.InsertIntoTable.Catalog,
				Schema:  q.InsertIntoTable.Schema,
				Name:    q.InsertIntoTable.Name,
			}
		}
		out = append(out, &plugin.Query{
			Name:            q.Metadata.Name,
			Cmd:             q.Metadata.Cmd,
			Text:            q.SQL,
			Comments:        q.Metadata.Comments,
			Columns:         columns,
			ExcludedColumns: excludedColumns,
			Params:          params,
			Filename:        q.Metadata.Filename,
			InsertIntoTable: iit,
		})
	}
	return out
}

func pluginQueryColumn(c *compiler.Column) *plugin.Column {
	l := -1
	if c.Length != nil {
		l = *c.Length
	}
	out := &plugin.Column{
		Name:         c.Name,
		OriginalName: c.OriginalName,
		Comment:      c.Comment,
		NotNull:      c.NotNull,
		Unsigned:     c.Unsigned,
		IsArray:      c.IsArray,
		ArrayDims:    int32(c.ArrayDims),
		Length:       int32(l),
		IsNamedParam: c.IsNamedParam,
		IsFuncCall:   c.IsFuncCall,
		IsSqlcSlice:  c.IsSqlcSlice,
	}

	if c.Type != nil {
		out.Type = &plugin.Identifier{
			Catalog: c.Type.Catalog,
			Schema:  c.Type.Schema,
			Name:    c.Type.Name,
		}
		out.TypeMods = pluginTypeMods(c.Type.Typmods)
	} else {
		out.Type = &plugin.Identifier{
			Name: c.DataType,
		}
	}

	if c.Table != nil {
		out.Table = &plugin.Identifier{
			Catalog: c.Table.Catalog,
			Schema:  c.Table.Schema,
			Name:    c.Table.Name,
		}
	}

	if c.EmbedTable != nil {
		out.EmbedTable = &plugin.Identifier{
			Catalog: c.EmbedTable.Catalog,
			Schema:  c.EmbedTable.Schema,
			Name:    c.EmbedTable.Name,
		}
	}

	return out
}

func pluginQueryParam(p compiler.Parameter) *plugin.Parameter {
	return &plugin.Parameter{
		Number: int32(p.Number),
		Column: pluginQueryColumn(p.Column),
	}
}

func codeGenRequest(r *compiler.Result, settings config.CombinedSettings) *plugin.GenerateRequest {
	return &plugin.GenerateRequest{
		Settings:    pluginSettings(r, settings),
		Catalog:     pluginCatalog(r.Catalog),
		Queries:     pluginQueries(r),
		SqlcVersion: info.Version,
	}
}
