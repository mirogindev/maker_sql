package mclause

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"strings"
)

type Column struct {
	Name     string
	Alias    string
	Function string
	Coalesce bool
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
	if _, ok := s.ColumnsMap[col.Name]; col.Function == "" && ok {
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
	baseTable := gstm.Schema.Table
	baseTableWithLevel := fmt.Sprintf("%s%v", baseTable, s.Level)

	builder.WriteString("SELECT ")
	if len(s.Columns) > 0 {

		if s.Distinct {
			builder.WriteString("DISTINCT ")
		}

		for idx, column := range s.Columns {
			var f *schema.Field
			if column.Name == "~~~py~~~" {
				f = gstm.Schema.PrimaryFields[0]
			} else {
				f = gstm.Schema.FieldsByDBName[column.Name]
			}
			if f == nil {
				logrus.Errorf("db field with name %s not found \n", column.Name)
				continue
			}

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
				closeSym := ")"
				if column.Coalesce {
					builder.WriteString(fmt.Sprintf(" COALESCE(%s(", column.Function))
					closeSym = "),0)"
				} else {
					builder.WriteString(fmt.Sprintf(" %s(", column.Function))
				}

				builder.WriteQuoted(gc)
				builder.WriteString(fmt.Sprintf("%s AS \"%s\"", closeSym, alias))
			} else {
				builder.WriteQuoted(gc)
			}
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
