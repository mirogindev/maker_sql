package callbacks

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/mirogindev/maker_sql/mclause"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"log"
	"reflect"
	"sort"
	"strings"
)

func BeforeQuery(db *gorm.DB) {
	if c, ok := db.Statement.Clauses[mclause.JSON_BUILD]; ok {
		jb := c.Expression.(*mclause.JsonBuild)
		level := jb.Level
		baseTable := db.Statement.Table
		baseTableAlias := fmt.Sprintf("%s%v", strings.Title(baseTable), level)
		jb.BaseTable = baseTable
		jb.BaseTableAlias = baseTableAlias
		joins := prepareQuery(db.Statement, baseTableAlias, level)
		if len(joins) > 0 {
			sj := sortJoins(joins)

			curJoinsMap := make(map[string]string)
			if j := db.Statement.Joins; j != nil {
				for _, k := range j {
					curJoinsMap[k.Name] = k.Name
				}
			}

			for _, j := range sj {
				if _, ok2 := curJoinsMap[j]; !ok2 {
					db = db.Joins(j)
				}
			}
		}
	}
}

func Query(db *gorm.DB) {
	if db.Error == nil {
		BuildQuerySQL(db)

		if !db.DryRun && db.Error == nil {
			rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
			if err != nil {
				db.AddError(err)
				return
			}
			gorm.Scan(rows, db, 0)
			db.AddError(rows.Close())
		}
	}
}

func BuildQuerySQL(db *gorm.DB) {
	var level *int
	baseTable := clause.CurrentTable
	baseTableAlias := baseTable

	if c, ok := db.Statement.Clauses[mclause.JSON_BUILD]; ok {
		jb := c.Expression.(*mclause.JsonBuild)
		level = &jb.Level
		baseTableAlias = fmt.Sprintf("\"%s%v\"", strings.Title(db.Statement.Table), jb.Level)
		db.Statement.TableExpr = &clause.Expr{
			SQL: fmt.Sprintf("%s %s", db.Statement.Table, baseTableAlias),
		}

	}
	if db.Statement.Schema != nil {
		for _, c := range db.Statement.Schema.QueryClauses {
			db.Statement.AddClause(c)
		}
	}

	if db.Statement.SQL.Len() == 0 {
		db.Statement.SQL.Grow(100)
		clauseSelect := clause.Select{Distinct: db.Statement.Distinct}

		if db.Statement.ReflectValue.Kind() == reflect.Struct && db.Statement.ReflectValue.Type() == db.Statement.Schema.ModelType {
			var conds []clause.Expression
			for _, primaryField := range db.Statement.Schema.PrimaryFields {
				if v, isZero := primaryField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); !isZero {
					conds = append(conds, clause.Eq{Column: clause.Column{Table: db.Statement.Table, Name: primaryField.DBName}, Value: v})
				}
			}

			if len(conds) > 0 {
				db.Statement.AddClause(clause.Where{Exprs: conds})
			}
		}

		if len(db.Statement.Selects) > 0 {
			clauseSelect.Columns = make([]clause.Column, len(db.Statement.Selects))
			for idx, name := range db.Statement.Selects {
				if db.Statement.Schema == nil {
					clauseSelect.Columns[idx] = clause.Column{Name: name, Raw: true}
				} else if f := db.Statement.Schema.LookUpField(name); f != nil {
					clauseSelect.Columns[idx] = clause.Column{Name: f.DBName}
				} else {
					clauseSelect.Columns[idx] = clause.Column{Name: name, Raw: true}
				}
			}
		} else if db.Statement.Schema != nil && len(db.Statement.Omits) > 0 {
			selectColumns, _ := db.Statement.SelectAndOmitColumns(false, false)
			clauseSelect.Columns = make([]clause.Column, len(db.Statement.Schema.DBNames))
			for _, dbName := range db.Statement.Schema.DBNames {
				if v, ok := selectColumns[dbName]; (ok && v) || !ok {
					clauseSelect.Columns = append(clauseSelect.Columns, clause.Column{Table: db.Statement.Table, Name: dbName})
				}
			}
		} else if db.Statement.Schema != nil && db.Statement.ReflectValue.IsValid() {
			queryFields := db.QueryFields
			if !queryFields {
				switch db.Statement.ReflectValue.Kind() {
				case reflect.Struct:
					queryFields = db.Statement.ReflectValue.Type() != db.Statement.Schema.ModelType
				case reflect.Slice:
					queryFields = db.Statement.ReflectValue.Type().Elem() != db.Statement.Schema.ModelType
				}
			}

			if queryFields {
				stmt := gorm.Statement{DB: db}
				// smaller struct
				if err := stmt.Parse(db.Statement.Dest); err == nil && (db.QueryFields || stmt.Schema.ModelType != db.Statement.Schema.ModelType) {
					clauseSelect.Columns = make([]clause.Column, len(stmt.Schema.DBNames))

					for idx, dbName := range stmt.Schema.DBNames {
						clauseSelect.Columns[idx] = clause.Column{Table: db.Statement.Table, Name: dbName}
					}
				}
			}
		}

		// inline joins
		joins := []clause.Join{}
		if fromClause, ok := db.Statement.Clauses["FROM"].Expression.(clause.From); ok {
			joins = fromClause.Joins
		}

		if len(db.Statement.Joins) != 0 || len(joins) != 0 {
			if len(db.Statement.Selects) == 0 && len(db.Statement.Omits) == 0 && db.Statement.Schema != nil {
				clauseSelect.Columns = make([]clause.Column, len(db.Statement.Schema.DBNames))
				for idx, dbName := range db.Statement.Schema.DBNames {
					clauseSelect.Columns[idx] = clause.Column{Table: db.Statement.Table, Name: dbName}
				}
			}

			for _, join := range db.Statement.Joins {
				splJoin := strings.Split(join.Name, ".")

				if db.Statement.Schema == nil {
					joins = append(joins, clause.Join{
						Expression: clause.NamedExpr{SQL: join.Name, Vars: join.Conds},
					})
				} else if relation, path, ok := findRelation(db.Statement.Schema.Relationships.Relations, splJoin); ok {

					if relation.Type == schema.Many2Many {
						targetTableAliasName := getAlias(join.Name, level)
						mmTableAliasName := getAlias(relation.JoinTable.Table, level)

						mmExprs := make([]clause.Expression, 0)
						exprs := make([]clause.Expression, 0)
						for _, ref := range relation.References {
							if ref.OwnPrimaryKey {
								mmExprs = append(mmExprs, clause.Eq{
									Column: clause.Column{Table: baseTableAlias, Name: ref.PrimaryKey.DBName},
									Value:  clause.Column{Table: mmTableAliasName, Name: ref.ForeignKey.DBName},
								})
							} else {
								if ref.PrimaryValue == "" {
									exprs = append(exprs, clause.Eq{
										Column: clause.Column{Table: mmTableAliasName, Name: ref.ForeignKey.DBName},
										Value:  clause.Column{Table: targetTableAliasName, Name: ref.PrimaryKey.DBName},
									})

								} else {
									exprs = append(exprs, clause.Eq{
										Column: clause.Column{Table: targetTableAliasName, Name: ref.ForeignKey.DBName},
										Value:  ref.PrimaryValue,
									})
								}
							}
						}

						joins = append(joins, clause.Join{
							Table: clause.Table{Name: relation.JoinTable.Table, Alias: mmTableAliasName},
							ON:    clause.Where{Exprs: mmExprs},
						})
						joins = append(joins, clause.Join{
							Table: clause.Table{Name: relation.FieldSchema.Table, Alias: targetTableAliasName},
							ON:    clause.Where{Exprs: exprs},
						})
					} else {
						tableAliasName := getAlias(join.Name, level)

						exprs := make([]clause.Expression, len(relation.References))
						for idx, ref := range relation.References {

							if ref.OwnPrimaryKey {
								exprs[idx] = clause.Eq{
									Column: clause.Column{Table: getAlias(ref.PrimaryKey.Schema.Table, level), Name: ref.PrimaryKey.DBName},
									Value:  clause.Column{Table: getAlias(join.Name, level), Name: ref.ForeignKey.DBName},
								}
							} else {
								foreignTableAlias := ""
								if ref.ForeignKey.Schema.Table != db.Statement.Table {
									foreignTableAlias = getAlias(path, level)
								} else {
									foreignTableAlias = getAlias(ref.ForeignKey.Schema.Table, level)
								}
								if ref.PrimaryValue == "" {
									exprs[idx] = clause.Eq{
										Column: clause.Column{Table: foreignTableAlias, Name: ref.ForeignKey.DBName},
										Value:  clause.Column{Table: getAlias(join.Name, level), Name: ref.PrimaryKey.DBName},
									}
								} else {
									exprs[idx] = clause.Eq{
										Column: clause.Column{Table: foreignTableAlias, Name: ref.ForeignKey.DBName},
										Value:  ref.PrimaryValue,
									}
								}
							}
						}

						onStmt := gorm.Statement{Table: tableAliasName, DB: db, Clauses: map[string]clause.Clause{}}
						for _, c := range relation.FieldSchema.QueryClauses {
							onStmt.AddClause(c)
						}

						if join.On != nil {
							onStmt.AddClause(join.On)
						}

						if cs, ok := onStmt.Clauses["WHERE"]; ok {
							if where, ok := cs.Expression.(clause.Where); ok {
								where.Build(&onStmt)

								if onSQL := onStmt.SQL.String(); onSQL != "" {
									vars := onStmt.Vars
									for idx, v := range vars {
										bindvar := strings.Builder{}
										onStmt.Vars = vars[0 : idx+1]
										db.Dialector.BindVarTo(&bindvar, &onStmt, v)
										onSQL = strings.Replace(onSQL, bindvar.String(), "?", 1)
									}

									exprs = append(exprs, clause.Expr{SQL: onSQL, Vars: vars})
								}
							}
						}
						joins = append(joins, clause.Join{
							Type:  clause.LeftJoin,
							Table: clause.Table{Name: relation.FieldSchema.Table, Alias: tableAliasName},
							ON:    clause.Where{Exprs: exprs},
						})
					}

				} else {
					joins = append(joins, clause.Join{
						Expression: clause.NamedExpr{SQL: join.Name, Vars: join.Conds},
					})
				}
			}

			db.Statement.AddClause(clause.From{Joins: joins})
			db.Statement.Joins = nil
		} else {
			db.Statement.AddClauseIfNotExists(clause.From{})
		}

		db.Statement.AddClauseIfNotExists(clauseSelect)

		db.Statement.Build(db.Statement.BuildClauses...)
	}
}

func findRelation(rels map[string]*schema.Relationship, arr []string) (*schema.Relationship, string, bool) {
	var path string
	for i, r := range arr {
		if rel, ok := rels[r]; ok {
			if len(arr) > 1 {
				path = r
				iRel, iPath, iOk := findRelation(rel.FieldSchema.Relationships.Relations, arr[i+1:])
				if iPath != "" {
					path = fmt.Sprintf("%s.%s", path, iPath)
				}

				return iRel, path, iOk
			}
			return rel, path, true
		}
	}
	return nil, "", false
}

func getAlias(name string, level *int) string {

	if level != nil {
		return fmt.Sprintf("\"%s%v\"", strings.Title(name), *level)
	}

	return name
}

func prepareQuery(st *gorm.Statement, tableAlias string, level int) map[string]string {
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

	joins := make(map[string]string)

	if wh, ok := jExpr.(clause.Where); ok {
		wh.Exprs = preprocessWhereClause(wh.Exprs, tableAlias, level, joins)
		whereClause.Expression = wh
		cl[WhereName] = whereClause
	}

	if gb, ok := gExpr.(clause.GroupBy); ok {
		gb.Columns = preprocessGroupBYClause(gb.Columns, tableAlias, level, joins)
		groupBy.Expression = gb
		cl[GroupByName] = groupBy
	}

	if gb, ok := oExpr.(clause.OrderBy); ok {
		gb.Columns = preprocessOrderBYClause(gb.Columns, tableAlias, level, joins)
		orderBy.Expression = gb
		cl[OrderByName] = orderBy
	}
	return joins
}

func preprocessOrderBYClause(cols []clause.OrderByColumn, tableAlias string, level int, joins map[string]string) []clause.OrderByColumn {
	tableAlias = fmt.Sprintf("%s", tableAlias)
	for i, v := range cols {
		spl := strings.Split(v.Column.Name, " ")
		var join []string
		v.Column.Name, join = replaceTableNamesWIthLevel(spl[0], tableAlias, level)
		if len(spl) > 1 {
			v.Column.Name = fmt.Sprintf("%s %s", v.Column.Name, spl[1])
		}
		cols[i] = v
		if join != nil {
			for _, j := range join {
				joins[j] = j
			}

		}
	}

	return cols
}

func preprocessGroupBYClause(cols []clause.Column, tableAlias string, level int, joins map[string]string) []clause.Column {
	for i, v := range cols {
		var join []string
		v.Name, join = replaceTableNamesWIthLevel(v.Name, tableAlias, level)
		cols[i] = v
		if join != nil {
			for _, j := range join {
				joins[j] = j
			}

		}
	}

	return cols
}

func sortJoins(joins map[string]string) []string {
	arr := make([]string, len(joins))
	counter := 0
	for i, _ := range joins {
		arr[counter] = i
		counter++
	}

	sort.Slice(arr, func(i, j int) bool {
		return len(strings.Split(arr[i], ".")) < len(strings.Split(arr[j], "."))
	})

	return arr
}

func preprocessWhereClause(exprs []clause.Expression, tableAlias string, level int, joins map[string]string) []clause.Expression {

	for i, v := range exprs {
		var join []string
		if ce, ok := v.(clause.Expr); ok {
			ce.SQL, join = replaceTableNamesWIthLevel(ce.SQL, tableAlias, level)
			exprs[i] = ce
		} else if ne, ok := v.(clause.NamedExpr); ok {
			ne.SQL, join = replaceTableNamesWIthLevel(ne.SQL, tableAlias, level)
			exprs[i] = ne
		} else if oc, ok := v.(clause.OrConditions); ok {
			oc.Exprs = preprocessWhereClause(oc.Exprs, tableAlias, level, joins)
			exprs[i] = oc
		} else if ac, ok := v.(clause.AndConditions); ok {
			ac.Exprs = preprocessWhereClause(ac.Exprs, tableAlias, level, joins)
			exprs[i] = ac
		} else {
			log.Println("Invalid type %T", v)
		}
		if join != nil {
			for _, j := range join {
				joins[j] = j
			}

		}
	}
	return exprs
}

func replaceTableNamesWIthLevel(_sql string, tableAlias string, level int) (string, []string) {
	s := strings.Split(_sql, ".")
	ln := len(s)
	if ln < 2 {
		return fmt.Sprintf("\"%s\".%s", tableAlias, _sql), nil
	}
	fn := s[ln-1]
	var sb strings.Builder
	joins := make([]string, 0)
	for i, s := range s[0 : ln-1] {
		if i > 0 {
			sb.WriteByte('.')
		}
		n := strcase.ToCamel(s)
		sb.WriteString(n)
		joins = append(joins, sb.String())
	}
	rp := fmt.Sprintf("\"%s%v\".%s", sb.String(), level, fn)

	return rp, joins
}
