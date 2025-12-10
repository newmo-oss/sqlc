package compiler

import (
	"context"
	"fmt"

	"github.com/sqlc-dev/sqlc/internal/analyzer"
	"github.com/sqlc-dev/sqlc/internal/config"
	"github.com/sqlc-dev/sqlc/internal/dbmanager"
	"github.com/sqlc-dev/sqlc/internal/engine/dolphin"
	"github.com/sqlc-dev/sqlc/internal/engine/postgresql"
	pganalyze "github.com/sqlc-dev/sqlc/internal/engine/postgresql/analyzer"
	"github.com/sqlc-dev/sqlc/internal/engine/sqlite"
	sqliteanalyze "github.com/sqlc-dev/sqlc/internal/engine/sqlite/analyzer"
	"github.com/sqlc-dev/sqlc/internal/opts"
	"github.com/sqlc-dev/sqlc/internal/sql/ast"
	"github.com/sqlc-dev/sqlc/internal/sql/catalog"
)

type Compiler struct {
	conf          config.SQL
	combo         config.CombinedSettings
	catalog       *catalog.Catalog
	parser        Parser
	result        *Result
	analyzer      analyzer.Analyzer
	client        dbmanager.Client
	selector      selector
	excludeFilter *ExcludeColumnsFilter

	schema []string
}

func NewCompiler(conf config.SQL, combo config.CombinedSettings) (*Compiler, error) {
	c := &Compiler{
		conf:          conf,
		combo:         combo,
		excludeFilter: NewExcludeColumnsFilter(conf.ExcludeColumns),
	}

	if conf.Database != nil && conf.Database.Managed {
		client := dbmanager.NewClient(combo.Global.Servers)
		c.client = client
	}

	switch conf.Engine {
	case config.EngineSQLite:
		c.parser = sqlite.NewParser()
		c.catalog = sqlite.NewCatalog()
		c.selector = newSQLiteSelector()
		if conf.Database != nil {
			if conf.Analyzer.Database == nil || *conf.Analyzer.Database {
				c.analyzer = analyzer.Cached(
					sqliteanalyze.New(*conf.Database),
					combo.Global,
					*conf.Database,
				)
			}
		}
	case config.EngineMySQL:
		c.parser = dolphin.NewParser()
		c.catalog = dolphin.NewCatalog()
		c.selector = newDefaultSelector()
	case config.EnginePostgreSQL:
		c.parser = postgresql.NewParser()
		c.catalog = postgresql.NewCatalog()
		c.selector = newDefaultSelector()
		if conf.Database != nil {
			if conf.Analyzer.Database == nil || *conf.Analyzer.Database {
				c.analyzer = analyzer.Cached(
					pganalyze.New(c.client, *conf.Database),
					combo.Global,
					*conf.Database,
				)
			}
		}
	default:
		return nil, fmt.Errorf("unknown engine: %s", conf.Engine)
	}
	return c, nil
}

func (c *Compiler) Catalog() *catalog.Catalog {
	return c.catalog
}

func (c *Compiler) ParseCatalog(schema []string) error {
	return c.parseCatalog(schema)
}

func (c *Compiler) ParseQueries(queries []string, o opts.Parser) error {
	r, err := c.parseQueries(o)
	if err != nil {
		return err
	}
	c.result = r
	return nil
}

func (c *Compiler) Result() *Result {
	return c.result
}

func (c *Compiler) Close(ctx context.Context) {
	if c.analyzer != nil {
		c.analyzer.Close(ctx)
	}
	if c.client != nil {
		c.client.Close(ctx)
	}
}

// MarkExcludedColumns marks columns as excluded based on exclude_columns config.
// This should be called after ParseCatalog to set IsExcluded flag on catalog columns.
func (c *Compiler) MarkExcludedColumns() {
	if c.excludeFilter == nil || len(c.conf.ExcludeColumns) == 0 {
		return
	}

	for _, schema := range c.catalog.Schemas {
		for _, table := range schema.Tables {
			tableName := &ast.TableName{
				Catalog: table.Rel.Catalog,
				Schema:  table.Rel.Schema,
				Name:    table.Rel.Name,
			}
			for _, col := range table.Columns {
				if c.excludeFilter.ShouldExclude(tableName, col.Name) {
					col.IsExcluded = true
				}
			}
		}
	}
}
