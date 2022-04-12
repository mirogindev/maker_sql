package mclause

import (
	"fmt"
	"gorm.io/gorm"
	clauses "gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"log"
	"regexp"
	"sqlgenerator"
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

type AggrQuery struct {
	Type   string
	Fields []string
}

type Field struct {
	Name       string
	AggrQuery  *AggrQuery
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

	//	GROUP_BY := "GROUP BY"
	selectClause := stmt.Clauses[SELECT]
	//	fs := stmt.Clauses["FOR"]
	//	groupByClause := stmt.Clauses[GROUP_BY]

	if selectClause.BeforeExpression == nil {
		selectClause.BeforeExpression = &s
	}

	//if fs.BeforeExpression == nil {
	//	fs.Expression = &s
	//}
	//if groupByClause.BeforeExpression == nil {
	//	groupByClause.BeforeExpression = &GroupByHelper{}
	//}

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

	fc := gstm.Clauses["FOR"]
	fc.Expression = s
	gstm.Clauses["FOR"] = fc

	if s.Level > 0 {
		s.GenerateFieldJoins(gstm)
	}

	if s.JsonAgg {
		builder.WriteString("SELECT json_agg(json_build_object(")
	} else {
		builder.WriteString("SELECT json_build_object(")
	}
	baseTable := gstm.Schema.Table
	baseTableAlias := fmt.Sprintf("%s%v", strings.Title(baseTable), s.Level)

	preprocessQuery(gstm, baseTableAlias, s.Level)
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
							Alias: fmt.Sprintf("%s%v", strings.Title(relation.JoinTable.Table), level),
						},
					}

					sql := query.Table(
						fmt.Sprintf("%s \"%s\"", relation.FieldSchema.Table,
							fmt.Sprintf("%s%v", relation.Name, level),
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
						st.AddColumn(Column{Name: foreignKeyName})
					}

					sql := query.Table(
						fmt.Sprintf("%s %s", relation.FieldSchema.Table,
							fmt.Sprintf("\"%s%v\"", strings.Title(relation.FieldSchema.Table), level),
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
							fmt.Sprintf("\"%s%v\"", strings.Title(relation.FieldSchema.Table), level),
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
			} else if column.AggrQuery != nil {
				aggrQuery := column.AggrQuery

				statement := gstm.Statement
				selectClauses := statement.Clauses["SELECT"]
				selectExpression := selectClauses.Expression.(*Select)
				groupByClause := gstm.Clauses["GROUP BY"]
				groupByColumns := make([]clauses.Column, len(aggrQuery.Fields)-1)

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
					})
				}
				for _, c := range s.Fields {
					if c.Query == nil && c.AggrQuery == nil {
						alias := fmt.Sprintf("%s%v_%s", baseTable, s.Level, c.Name)
						groupByColumns = append(groupByColumns, clauses.Column{Name: alias})
					}
				}

				groupByClause.BeforeExpression = GroupByHelper{}
				groupByClause.Expression = clauses.GroupBy{
					Columns: groupByColumns,
				}
				gstm.Clauses["GROUP BY"] = groupByClause

				builder.WriteByte(')')

			} else {
				f := gstm.Schema.FieldsByName[sqlgenerator.ToCamelCase(column.Name)]
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

	//for _, event := range []string{"LIMIT", "ORDER BY", "GROUP BY", "WHERE", "FROM"} {
	//	if cl, ok := gstm.Clauses[event]; ok {
	//		cl.AfterExpression = s
	//		gstm.Clauses[event] = cl
	//		break
	//	}
	//}

}

func preprocessQuery(st *gorm.Statement, tableAlias string, level int) {
	WhereName := "WHERE"
	GroupByName := "GROUP BY"
	OrderByName := "ORDER BY"

	cl := st.Clauses
	whereClause := cl[WhereName]
	groupBy := cl[GroupByName]
	orderBy := cl[OrderByName]
	jExpr := whereClause.Expression
	gExpr := groupBy.Expression
	oExpr := orderBy.Expression
	if wh, ok := jExpr.(clauses.Where); ok {
		wh.Exprs = preprocessWhereClause(wh.Exprs, level)
		whereClause.Expression = wh
		cl[WhereName] = whereClause
	}

	if gb, ok := gExpr.(clauses.GroupBy); ok {
		gb.Columns = preprocessGroupBYClause(gb.Columns, tableAlias, level)
		groupBy.Expression = gb
		cl[GroupByName] = groupBy
	}

	if gb, ok := oExpr.(clauses.OrderBy); ok {
		gb.Columns = preprocessOrderBYClause(gb.Columns, tableAlias, level)
		orderBy.Expression = gb
		cl[OrderByName] = orderBy
	}
}

func preprocessOrderBYClause(cols []clauses.OrderByColumn, tableAlias string, level int) []clauses.OrderByColumn {
	tableAlias = fmt.Sprintf("\"%s\"", tableAlias)
	for i, v := range cols {
		spl := strings.Split(v.Column.Name, " ")
		v.Column.Name = replaceColumnNamesWIthLevel(spl[0], tableAlias, level)
		if len(spl) > 1 {
			v.Column.Name = fmt.Sprintf("%s %s", v.Column.Name, spl[1])
		}

		cols[i] = v
	}

	return cols
}

func preprocessGroupBYClause(cols []clauses.Column, tableAlias string, level int) []clauses.Column {
	for i, v := range cols {
		v.Name = replaceColumnNamesWIthLevel(v.Name, tableAlias, level)
		cols[i] = v
	}

	return cols
}

func preprocessWhereClause(exprs []clauses.Expression, level int) []clauses.Expression {
	for i, v := range exprs {
		if ce, ok := v.(clauses.Expr); ok {
			ce.SQL = replaceTableNamesWIthLevel(ce.SQL, level)
			exprs[i] = ce
		} else if ce, ok := v.(clauses.NamedExpr); ok {
			ce.SQL = replaceTableNamesWIthLevel(ce.SQL, level)
			exprs[i] = ce
		} else if ce, ok := v.(clauses.OrConditions); ok {
			ce.Exprs = preprocessWhereClause(ce.Exprs, level)
			exprs[i] = ce
		} else if ce, ok := v.(clauses.AndConditions); ok {
			ce.Exprs = preprocessWhereClause(ce.Exprs, level)
			exprs[i] = ce
		} else {
			log.Println("Invalid type %T", v)
		}

	}
	return exprs
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

func replaceTableNamesWIthLevel(_sql string, level int) string {
	var re = regexp.MustCompile(`"(.*?)"`)
	s := re.ReplaceAllString(_sql, strings.ToLower(fmt.Sprintf(`"${1}%v"`, level)))
	return s
}

func replaceColumnNamesWIthLevel(_sql string, tableAlias string, level int) string {
	spl := strings.Split(_sql, ".")
	if len(spl) == 1 {
		return fmt.Sprintf("%s.%s", tableAlias, _sql)
	}

	var re = regexp.MustCompile(`"(.*?)"`)
	s := re.ReplaceAllString(_sql, strings.ToLower(fmt.Sprintf(`"${1}%v"`, level)))
	return s
}
