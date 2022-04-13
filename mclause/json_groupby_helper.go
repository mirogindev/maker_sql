package mclause

import (
	clauses "gorm.io/gorm/clause"
)

type GroupByHelper struct {
}

func (s GroupByHelper) Build(builder clauses.Builder) {
	builder.WriteString("GROUP BY ")
}
