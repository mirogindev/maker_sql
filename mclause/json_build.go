package mclause

import (
	"fmt"
	"gorm.io/gorm"
	clauses "gorm.io/gorm/clause"
	"log"
	"sqlgenerator"
)

type Field struct {
	Name       string
	Path       string
	Query      *gorm.DB
	TargetType interface{}
}

type JsonBuild struct {
	Level        int
	ParentType   interface{}
	initialized  bool
	JsonAgg      bool
	Fields       []Field
	SelectClause *clauses.Select
	Expression   clauses.Expression
}

func (s JsonBuild) ModifyStatement(stmt *gorm.Statement) {

	SELECT := "SELECT"
	clause := stmt.Clauses[SELECT]

	if clause.BeforeExpression == nil {
		clause.BeforeExpression = &s
	}

	colums := []clauses.Column{}
	for _, c := range s.Fields {
		if c.Query != nil {
			continue
		}
		colums = append(colums, clauses.Column{
			Name: c.Name,
		})
	}

	sc := &Select{Columns: colums, Level: s.Level}
	clause.Expression = sc
	stmt.Clauses[SELECT] = clause
}

func (s JsonBuild) Build(builder clauses.Builder) {
	if s.initialized {
		if s.Level == 0 {
			builder.WriteString(") as root")
		}
		return
	}
	s.initialized = true

	gstm := builder.(*gorm.Statement)

	for _, event := range []string{"LIMIT", "ORDER BY", "WHERE"} {
		if cl, ok := gstm.Clauses[event]; ok {
			cl.AfterExpression = s
			gstm.Clauses[event] = cl
			break
		}
	}

	if s.JsonAgg {
		builder.WriteString("SELECT json_agg(json_build_object(")
	} else {
		builder.WriteString("SELECT json_build_object(")
	}

	if len(s.Fields) > 0 {
		baseTable := gstm.Schema.Table
		for idx, column := range s.Fields {
			f := gstm.Schema.FieldsByName[sqlgenerator.ToCamelCase(column.Name)]
			if f == nil {
				log.Fatalf("Field with name %s is not found", sqlgenerator.ToCamelCase(column.Name))
				continue
			}

			if idx > 0 {
				builder.WriteByte(',')
				builder.WriteByte('\n')
			} else {
				builder.WriteByte('\n')
			}
			builder.WriteByte('\'')
			builder.WriteString(column.Name)
			builder.WriteByte('\'')
			builder.WriteByte(',')
			if column.Query != nil {
				query := column.Query
				statement := column.Query.Statement
				selectClauses := statement.Clauses["SELECT"]

				builder.WriteByte('\n')
				builder.WriteByte('(')
				if _, ok := f.TagSettings["MANY2MANY"]; ok {
					level := s.Level + 1
					se := selectClauses.Expression.(*Select)
					se.Level = level
					jb := selectClauses.BeforeExpression.(*JsonBuild)
					jb.ParentType = gstm.Model
					jb.Level = level
					jb.JsonAgg = true
					//selectClauses.BeforeExpression = jb
					//selectClauses.Expression = se
					//statement.Clauses["SELECT"] = selectClauses
					sql := query.Find(column.TargetType).Statement.SQL.String()
					builder.WriteString(sql)
					builder.WriteString(") as root")
					builder.WriteString(")")
				}

				builder.WriteByte('\n')
			} else {
				alias := fmt.Sprintf("%s%v_%s", baseTable, s.Level, f.DBName)
				builder.WriteString(alias)
			}

		}

		builder.WriteByte(')')

		if s.JsonAgg {
			builder.WriteByte(')')
		}
		builder.WriteString(" FROM ( ")
	} else {
		log.Fatalf("Json clause must have at least one field")
	}

}

func (s JsonBuild) MergeClause(clause *clauses.Clause) {
	if s.Expression != nil {
		clause.Expression = s.Expression
	} else {
		clause.Expression = s
	}
}
