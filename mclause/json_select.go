package mclause

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"strings"
)

type Column struct {
	Name     string
	Alias    string
	Function string
}

type Select struct {
	Level      int
	Distinct   bool
	Columns    []Column
	ColumnsMap map[string]Column
	Expression clause.Expression
}

func (s *Select) AddColumn(col Column) {
	if s.ColumnsMap == nil {
		s.ColumnsMap = make(map[string]Column)
	}
	if _, ok := s.ColumnsMap[col.Name]; ok {
		return
	}
	s.ColumnsMap[col.Name] = col
	s.Columns = append(s.Columns, col)
}

func (s *Select) ColumnNameExist(name string) bool {
	if s.ColumnsMap == nil {
		return false
	}
	if _, ok := s.ColumnsMap[name]; ok {
		return ok
	}

	return false
}

func (s Select) Name() string {
	return "SELECT"
}

func (s Select) Build(builder clause.Builder) {
	gstm := builder.(*gorm.Statement)
	fieldsMap := convertToCCMap(gstm.Schema.FieldsByName)
	baseTable := gstm.Schema.Table
	baseTableWithLevel := fmt.Sprintf("%s%v", baseTable, s.Level)

	builder.WriteString("SELECT ")
	if len(s.Columns) > 0 {

		if s.Distinct {
			builder.WriteString("DISTINCT ")
		}

		for idx, column := range s.Columns {
			f := fieldsMap[column.Name]

			alias := column.Alias
			if alias == "" {
				alias = fmt.Sprintf("%s%v_%s", baseTable, s.Level, f.DBName)
			}

			gc := clause.Column{}
			gc.Table = fmt.Sprintf("\"%s\"", strings.Title(baseTableWithLevel))

			if column.Function == "" {
				gc.Alias = alias
			}
			gc.Name = column.Name

			if idx > 0 {
				builder.WriteByte(',')
			}

			if column.Function != "" {
				builder.WriteString(fmt.Sprintf(" %s(", column.Function))
				builder.WriteQuoted(gc)
				builder.WriteString(fmt.Sprintf(") AS \"%s\"", alias))
			} else {
				builder.WriteQuoted(gc)
			}
		}
	} else {
		builder.WriteByte('*')
	}
}

func convertToCCMap(m map[string]*schema.Field) map[string]*schema.Field {
	res := make(map[string]*schema.Field)
	for k, v := range m {
		res[strcase.ToSnake(k)] = v
	}
	return res
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
