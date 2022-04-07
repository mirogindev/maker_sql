package mclause

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupBy struct {
	Level      int
	Columns    []clause.Column
	ColumnsMap map[string]clause.Column
	Expression clause.Expression
}

func (s *GroupBy) AddColumn(col clause.Column) {
	if s.ColumnsMap == nil {
		s.ColumnsMap = make(map[string]clause.Column)
	}
	s.ColumnsMap[col.Name] = col
	s.Columns = append(s.Columns, col)
}

func (s *GroupBy) ColumnNameExist(name string) bool {
	if s.ColumnsMap == nil {
		return false
	}
	if _, ok := s.ColumnsMap[name]; ok {
		return ok
	}

	return false
}

func (s GroupBy) Name() string {
	return "GROUP BY"
}

func (s GroupBy) Build(builder clause.Builder) {
	gstm := builder.(*gorm.Statement)
	baseTable := gstm.Schema.Table
	baseTableWithLevel := fmt.Sprintf("%s%v", baseTable, s.Level)

	builder.WriteString("GROUP BY ")
	if len(s.Columns) > 0 {
		for idx, column := range s.Columns {
			f := gstm.Schema.FieldsByDBName[column.Name]
			alias := fmt.Sprintf("%s%v_%s", baseTableWithLevel, s.Level, f.DBName)

			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(alias)
		}
	} else {
		builder.WriteByte('*')
	}
}

func (s GroupBy) MergeClause(c *clause.Clause) {
	if s.Expression != nil {
		c.Expression = s.Expression
	} else {
		c.Expression = s
	}
}
