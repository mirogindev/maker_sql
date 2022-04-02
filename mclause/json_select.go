package mclause

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Select struct {
	Level      int
	Distinct   bool
	Columns    []clause.Column
	Expression clause.Expression
}

func (s Select) Name() string {
	return "SELECT"
}

func (s Select) Build(builder clause.Builder) {
	gstm := builder.(*gorm.Statement)
	baseTable := gstm.Schema.Table

	builder.WriteString("SELECT ")
	if len(s.Columns) > 0 {

		if s.Distinct {
			builder.WriteString("DISTINCT ")
		}

		for idx, column := range s.Columns {
			f := gstm.Schema.FieldsByDBName[column.Name]
			alias := fmt.Sprintf("%s%v_%s", baseTable, s.Level, f.DBName)
			column.Alias = alias
			column.Table = baseTable

			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(column)
		}
	} else {
		builder.WriteByte('*')
	}
}

func (s Select) MergeClause(c *clause.Clause) {
	if s.Expression != nil {
		if s.Distinct {
			if expr, ok := s.Expression.(*clause.Expr); ok {
				expr.SQL = "DISTINCT " + expr.SQL
				c.Expression = expr
				return
			}
		}

		c.Expression = s.Expression
	} else {
		c.Expression = s
	}
}
