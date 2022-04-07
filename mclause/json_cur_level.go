package mclause

import (
	clauses "gorm.io/gorm/clause"
)

type JsonCurrentLevel struct {
	Level int
}

func (s JsonCurrentLevel) Build(builder clauses.Builder) {

}
