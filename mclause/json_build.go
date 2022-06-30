package mclause

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/mirogindev/maker_sql"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	clauses "gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"log"
	"reflect"
	"strings"
)

const (
	Sum   = "sum"
	Avg   = "avg"
	Max   = "max"
	Min   = "min"
	Count = "count"
)

var JSON_BUILD = "JSON_BUILD"
var SELECT = "SELECT"

type AggrQuery struct {
	Type   string
	Fields []string
}

type Field struct {
	Name      string
	AggrQuery *AggrQuery
	Query     *gorm.DB
}

type JsonBuild struct {
	BaseTable      string
	BaseTableAlias string
	Level          int
	FieldInfo      *schema.Field
	ParentType     interface{}
	initialized    bool
	JsonAgg        bool
	Fields         []Field
	Vars           []interface{}
	SelectClause   *clauses.Select
	Expression     clauses.Expression
}

func (s JsonBuild) ModifyStatement(stmt *gorm.Statement) {
	selectClause := stmt.Clauses[SELECT]

	if selectClause.BeforeExpression == nil {
		selectClause.BeforeExpression = &s
	}

	sc := &Select{Level: s.Level}
	for _, c := range s.Fields {
		if c.Query != nil || c.AggrQuery != nil {
			continue
		}
		sc.AddColumn(Column{Name: c.Name})
	}

	selectClause.Expression = sc
	stmt.Clauses[SELECT] = selectClause
	stmt.Clauses[JSON_BUILD] = clauses.Clause{Expression: &s}
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
	baseSelectClause := gstm.Clauses[SELECT]
	baseSelectExpression := baseSelectClause.Expression.(*Select)

	baseForClause := gstm.Clauses["FOR"]
	baseForClause.Expression = s

	gstm.Clauses["FOR"] = baseForClause

	if s.Level > 0 {
		s.GenerateFieldJoins(gstm)
	}

	if s.JsonAgg {
		builder.WriteString("SELECT json_agg(json_build_object(")
	} else {
		builder.WriteString("SELECT json_build_object(")
	}
	baseTable := s.BaseTable
	baseTableAlias := s.BaseTableAlias

	if len(s.Fields) > 0 {

		for idx, column := range s.Fields {

			if idx > 0 {
				builder.WriteByte(',')
				builder.WriteByte('\n')
			} else {
				builder.WriteByte('\n')
			}

			builder.WriteString(fmt.Sprintf("'%s'", column.Name))

			builder.WriteByte(',')
			if column.Query != nil {
				f := gstm.Schema.FieldsByName[sqlgenerator.ToCamelCase(column.Name)]

				if t, ok := f.Tag.Lookup("sql_gen"); ok {
					f = gstm.Schema.FieldsByName[sqlgenerator.ToCamelCase(t)]
				}
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
					targetType := reflect.New(f.FieldType.Elem().Elem()).Interface()
					selectExpression.Level = level

					jsonExpression.ParentType = gstm.Model
					jsonExpression.FieldInfo = f
					jsonExpression.Level = level
					jsonExpression.JsonAgg = true

					jc := clauses.Join{
						ON: clauses.Where{
							Exprs: s.buildJoinCondition(relation.References, baseTableAlias, baseSelectExpression),
						},
						Table: clauses.Table{
							Name:  relation.JoinTable.Table,
							Alias: fmt.Sprintf("%s%v", strings.Title(relation.JoinTable.Table), level),
						},
					}

					qstm := query.Session(&gorm.Session{DryRun: true}).Table(
						fmt.Sprintf("%s \"%s\"", relation.FieldSchema.Table,
							fmt.Sprintf("%s%v", relation.Name, level),
						),
					).Clauses(clauses.From{
						Joins: []clauses.Join{jc},
					}).Find(targetType).Statement

					gstm.Vars = append(gstm.Vars, qstm.Vars...)

					builder.WriteString(qstm.SQL.String())

					builder.WriteString(") as root")
					builder.WriteString(")")
				} else if relation.Type == schema.BelongsTo {
					targetType := reflect.New(f.FieldType.Elem()).Interface()
					primaryKeyName := relation.References[0].PrimaryKey.DBName
					foreignKeyName := relation.References[0].ForeignKey.DBName

					selectExpression.Level = level

					jsonExpression.ParentType = gstm.Model
					jsonExpression.FieldInfo = f
					jsonExpression.Level = level

					if !baseSelectExpression.ColumnNameExist(foreignKeyName) {
						baseSelectExpression.AddColumn(Column{Name: foreignKeyName})
					}

					qstm := query.Session(&gorm.Session{DryRun: true}).Table(
						fmt.Sprintf("%s %s", relation.FieldSchema.Table,
							fmt.Sprintf("\"%s%v\"", strings.Title(relation.FieldSchema.Table), level),
						),
					).Clauses(clauses.Where{
						Exprs: []clauses.Expression{
							clauses.NamedExpr{
								SQL: fmt.Sprintf("%s = %s_%s", primaryKeyName, baseTableAlias, foreignKeyName),
							},
						},
					}).Find(targetType).Statement

					gstm.Vars = append(gstm.Vars, qstm.Vars...)

					builder.WriteString(qstm.SQL.String())
					builder.WriteString(") as root")
					builder.WriteString(")")
				} else if relation.Type == schema.HasMany {
					targetType := reflect.New(f.FieldType.Elem().Elem()).Interface()
					primaryKeyName := relation.References[0].PrimaryKey.DBName
					foreignKeyName := relation.References[0].ForeignKey.DBName
					selectExpression.Level = level

					jsonExpression.ParentType = gstm.Model
					jsonExpression.FieldInfo = f
					jsonExpression.Level = level
					jsonExpression.JsonAgg = true

					if !baseSelectExpression.ColumnNameExist(primaryKeyName) {
						baseSelectExpression.AddColumn(Column{Name: primaryKeyName})
					}

					qstm := query.Session(&gorm.Session{DryRun: true}).Table(
						fmt.Sprintf("%s %s", relation.FieldSchema.Table,
							fmt.Sprintf("\"%s%v\"", strings.Title(relation.FieldSchema.Table), level),
						),
					).Clauses(clauses.Where{
						Exprs: []clauses.Expression{
							clauses.NamedExpr{
								SQL: fmt.Sprintf("%s = %s_%s", foreignKeyName, baseTableAlias, primaryKeyName),
							},
						},
					}).Find(targetType).Statement

					gstm.Vars = append(gstm.Vars, qstm.Vars...)

					builder.WriteString(qstm.SQL.String())

					builder.WriteString(") as root")
					builder.WriteString(")")
				}

				builder.WriteByte('\n')
			} else if column.AggrQuery != nil {
				aggrQuery := column.AggrQuery

				statement := gstm.Statement
				selectClauses := statement.Clauses["SELECT"]
				selectExpression := selectClauses.Expression.(*Select)
				groupByClause := gstm.Clauses["GROUP BY"]
				groupByColumns := make([]clauses.Column, 0, len(aggrQuery.Fields))

				builder.WriteString("json_build_object(")
				for i, ac := range aggrQuery.Fields {
					f := gstm.Schema.FieldsByName[sqlgenerator.ToCamelCase(ac)]
					alias := fmt.Sprintf("%s%v_%s", baseTable, s.Level, f.DBName)
					aliasWithFun := fmt.Sprintf("%s_%s", alias, aggrQuery.Type)
					if i > 0 {
						builder.WriteByte(',')
						builder.WriteByte('\n')
					} else {
						builder.WriteByte('\n')
					}

					builder.WriteString(fmt.Sprintf("'%s'", ac))
					builder.WriteByte(',')
					builder.WriteString(aliasWithFun)
					selectExpression.AddColumn(Column{
						Name:     ac,
						Alias:    aliasWithFun,
						Function: aggrQuery.Type,
						Coalesce: true,
					})
				}
				for _, c := range s.Fields {
					if c.AggrQuery == nil {
						if c.Query != nil {
							fr := gstm.Schema.Relationships.Relations[strcase.ToCamel(c.Name)]
							if fr == nil {
								logrus.Panicf("Reference field %s is not found", c.Name)
							}

							for _, l := range fr.References {
								var alias string

								if fr.Type == schema.Many2Many {
									alias = fmt.Sprintf("%s%v_%s", baseTable, s.Level, l.PrimaryKey.DBName)
								} else {
									alias = fmt.Sprintf("%s%v_%s", baseTable, s.Level, l.ForeignKey.DBName)
								}
								groupByColumns = append(groupByColumns, clauses.Column{Name: alias})
							}

						} else {
							alias := fmt.Sprintf("%s%v_%s", baseTable, s.Level, c.Name)
							groupByColumns = append(groupByColumns, clauses.Column{Name: alias})
						}

					}
				}

				groupByClause.BeforeExpression = GroupByHelper{}
				groupByClause.Expression = clauses.GroupBy{
					Columns: groupByColumns,
				}
				if len(groupByColumns) > 0 {
					gstm.Clauses["GROUP BY"] = groupByClause
				}

				builder.WriteByte(')')

			} else {
				f := gstm.Schema.FieldsByDBName[column.Name]
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

func (s JsonBuild) buildJoinCondition(references []*schema.Reference, baseTableAlias string, selectClause *Select) []clauses.Expression {
	if len(references) > 1 {
		andCond := clauses.AndConditions{}
		for _, r := range references {
			var exp clauses.NamedExpr
			if r.OwnPrimaryKey {
				if !selectClause.ColumnNameExist(r.PrimaryKey.DBName) {
					selectClause.AddColumn(Column{
						Name: r.PrimaryKey.DBName,
					})
				}
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
