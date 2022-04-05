package mclause

import (
	"fmt"
	"gorm.io/gorm"
	clauses "gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
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
	FieldInfo    *schema.Field
	ParentType   interface{}
	initialized  bool
	JsonAgg      bool
	Fields       []Field
	SelectClause *clauses.Select
	Expression   clauses.Expression
}

func (s JsonBuild) ModifyStatement(stmt *gorm.Statement) {

	SELECT := "SELECT"
	FROM := "FROM"
	selectClause := stmt.Clauses[SELECT]
	fromClause := stmt.Clauses[FROM]

	if selectClause.BeforeExpression == nil {
		selectClause.BeforeExpression = &s
	}

	sc := &Select{Level: s.Level}
	for _, c := range s.Fields {
		if c.Query != nil {
			continue
		}
		sc.AddColumn(clauses.Column{
			Name: c.Name,
		})
	}

	selectClause.Expression = sc
	stmt.Clauses[SELECT] = selectClause
	stmt.Clauses[FROM] = fromClause
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

	if s.Level > 0 {
		s.GenerateFieldJoins(gstm)
	}

	for _, event := range []string{"LIMIT", "ORDER BY", "WHERE", "FROM"} {
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
		baseTableAlias := fmt.Sprintf("%s%v", baseTable, s.Level)

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

			builder.WriteString(fmt.Sprintf("'%s'", column.Name))

			builder.WriteByte(',')
			if column.Query != nil {
				level := s.Level + 1
				query := column.Query
				statement := column.Query.Statement
				selectClauses := statement.Clauses["SELECT"]
				selectExpression := selectClauses.Expression.(*Select)
				jsonExpression := selectClauses.BeforeExpression.(*JsonBuild)
				relation := gstm.Schema.Relationships.Relations[f.Name]

				builder.WriteByte('\n')
				builder.WriteByte('(')

				if relation.Type == schema.Many2Many {
					selectExpression.Level = level

					jsonExpression.ParentType = gstm.Model
					jsonExpression.FieldInfo = f
					jsonExpression.Level = level
					jsonExpression.JsonAgg = true

					jc := clauses.Join{
						ON: clauses.Where{
							Exprs: s.buildJoinCondition(relation.References, baseTableAlias),
						},
						Table: clauses.Table{
							Name:  relation.JoinTable.Table,
							Alias: fmt.Sprintf("%s%v", relation.JoinTable.Table, level),
						},
					}

					sql := query.Table(
						fmt.Sprintf("%s %s", relation.FieldSchema.Table,
							fmt.Sprintf("%s%v", relation.FieldSchema.Table, level),
						),
					).Clauses(clauses.From{
						Joins: []clauses.Join{jc},
					}).Find(column.TargetType).Statement.SQL.String()

					builder.WriteString(sql)
					builder.WriteString(") as root")
					builder.WriteString(")")
				} else if relation.Type == schema.BelongsTo {
					primaryKeyName := relation.References[0].PrimaryKey.DBName
					foreignKeyName := relation.References[0].ForeignKey.DBName

					selectExpression.Level = level

					jsonExpression.ParentType = gstm.Model
					jsonExpression.FieldInfo = f
					jsonExpression.Level = level

					st := gstm.Clauses["SELECT"].Expression.(*Select)

					if !st.ColumnNameExist(foreignKeyName) {
						st.AddColumn(clauses.Column{
							Name: foreignKeyName,
						})
					}

					sql := query.Table(
						fmt.Sprintf("%s %s", relation.FieldSchema.Table,
							fmt.Sprintf("%s%v", relation.FieldSchema.Table, level),
						),
					).Clauses(clauses.Where{
						Exprs: []clauses.Expression{
							clauses.NamedExpr{
								SQL: fmt.Sprintf("%s = %s_%s", primaryKeyName, baseTableAlias, foreignKeyName),
							},
						},
					}).Find(column.TargetType).Statement.SQL.String()

					builder.WriteString(sql)
					builder.WriteString(") as root")
					builder.WriteString(")")
				} else if relation.Type == schema.HasMany {
					primaryKeyName := relation.References[0].PrimaryKey.DBName
					foreignKeyName := relation.References[0].ForeignKey.DBName
					selectExpression.Level = level

					jsonExpression.ParentType = gstm.Model
					jsonExpression.FieldInfo = f
					jsonExpression.Level = level
					jsonExpression.JsonAgg = true

					sql := query.Table(
						fmt.Sprintf("%s %s", relation.FieldSchema.Table,
							fmt.Sprintf("%s%v", relation.FieldSchema.Table, level),
						),
					).Clauses(clauses.Where{
						Exprs: []clauses.Expression{
							clauses.NamedExpr{
								SQL: fmt.Sprintf("%s = %s_%s", foreignKeyName, baseTableAlias, primaryKeyName),
							},
						},
					}).Find(column.TargetType).Statement.SQL.String()

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

func (s JsonBuild) GenerateFieldJoins(builder *gorm.Statement) {

}

func (s JsonBuild) buildJoinCondition(references []*schema.Reference, baseTableAlias string) []clauses.Expression {
	if len(references) > 1 {
		andCond := clauses.AndConditions{}
		//exprs := clauses.AndConditions{}
		for _, r := range references {
			var exp clauses.NamedExpr
			if r.OwnPrimaryKey {
				exp = clauses.NamedExpr{
					SQL: fmt.Sprintf("%s = %s_%s", r.ForeignKey.DBName, baseTableAlias, r.PrimaryKey.DBName),
				}
			} else {
				exp = clauses.NamedExpr{
					SQL: fmt.Sprintf("%s = %s", r.ForeignKey.DBName, r.PrimaryKey.DBName),
				}
			}
			andCond.Exprs = append(andCond.Exprs, exp)
		}
		return []clauses.Expression{andCond}
	}
	return nil
}

//		clauses.AndConditions{
//		Exprs: clauses.AndConditions{
//			clauses.NamedExpr{
//				SQL: fmt.Sprintf("tag_id = id"),
//			},
//			clauses.NamedExpr{
//				SQL: "user_id = users0_id",
//			},
//		},
//	},
//}
