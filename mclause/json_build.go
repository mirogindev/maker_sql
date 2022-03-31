package mclause

import (
	clauses "gorm.io/gorm/clause"
)

type Column struct {
	Name  string
	Path  string
	Query string
}

type JsonBuild struct {
	Columns    []Column
	Expression clauses.Expression
}

//func (s JsonBuild) ModifyStatement(stmt *gorm.Statement) {
//	log.Println(stmt)
//}

func (s JsonBuild) Name() string {
	return "SELECT"
}

func (s JsonBuild) Build(builder clauses.Builder) {
	builder.WriteString("json_build_object(")
	if len(s.Columns) > 0 {
		for idx, column := range s.Columns {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteByte('\'')
			builder.WriteString(column.Name)
			builder.WriteByte('\'')
			builder.WriteByte(',')
			builder.WriteString(column.Path)
		}
	}

	builder.WriteByte(')')
}

func (s JsonBuild) MergeClause(clause *clauses.Clause) {
	if s.Expression != nil {
		clause.Expression = s.Expression
	} else {
		clause.Expression = s
	}
}

// CommaExpression represents a group of expressions separated by commas.
type CommaExpression struct {
	Exprs []clauses.Expression
}

func (comma CommaExpression) Build(builder clauses.Builder) {
	for idx, expr := range comma.Exprs {
		if idx > 0 {
			_, _ = builder.WriteString(", ")
		}
		expr.Build(builder)
	}
}
