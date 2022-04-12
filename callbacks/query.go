package callbacks

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"reflect"
	"sqlgenerator/mclause"
	"strings"
)

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
				} else if relation, ok := findRelation(db.Statement.Schema.Relationships.Relations, splJoin); ok {

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
								if ref.PrimaryValue == "" {
									exprs[idx] = clause.Eq{
										Column: clause.Column{Table: getAlias(ref.ForeignKey.Schema.Table, level), Name: ref.ForeignKey.DBName},
										Value:  clause.Column{Table: getAlias(join.Name, level), Name: ref.PrimaryKey.DBName},
									}
								} else {
									exprs[idx] = clause.Eq{
										Column: clause.Column{Table: getAlias(ref.ForeignKey.Schema.Table, level), Name: ref.ForeignKey.DBName},
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

func findRelation(rels map[string]*schema.Relationship, arr []string) (*schema.Relationship, bool) {
	for i, r := range arr {
		if rel, ok := rels[r]; ok {
			if len(arr) > 1 {
				return findRelation(rel.FieldSchema.Relationships.Relations, arr[i+1:])
			}
			return rel, true
		}
	}
	return nil, false
}

func getAlias(name string, level *int) string {

	if level != nil {
		return fmt.Sprintf("\"%s%v\"", strings.Title(name), *level)
	}

	return name
}
