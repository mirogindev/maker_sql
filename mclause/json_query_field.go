package mclause

import (
	"gorm.io/gorm"
	clauses "gorm.io/gorm/clause"
	"log"
)

type JsonQueryField struct {
	Level      int
	Fields     []Field
	Expression clauses.Expression
}

func (s JsonQueryField) Name() string {
	return "SELECT22"
}

func (s JsonQueryField) ModifyStatement(stmt *gorm.Statement) {
	log.Println(stmt)
}

func (s JsonQueryField) Build(builder clauses.Builder) {

}

func (s JsonQueryField) MergeClause(clause *clauses.Clause) {
	if s.Expression != nil {
		clause.Expression = s.Expression
	} else {
		clause.Expression = s
	}
}
