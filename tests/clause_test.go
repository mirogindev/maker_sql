package tests

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
	"strings"
	"sync"
	"testing"
)

//func TestSelect(t *testing.T) {
//	results := []struct {
//		Clauses []clause.Interface
//		Result  string
//		Vars    []interface{}
//	}{
//		{
//			[]clause.Interface{mclause.JsonBuild{
//				Fields: []clause.Column{{Name: "id", Alias: "users.id"}, {Name: "name", Alias: "users.name"}},
//			}, clause.From{}},
//			"SELECT json_build_object('id',users.id,'name',users.name) FROM `users`", nil,
//		},
//		//{
//		//	[]clause.Interface{mclause.JsonBuild{
//		//		Fields: []clause.Column{clause.PrimaryColumn},
//		//	}, clause.From{}},
//		//	"SELECT `users`.`id` FROM `users`", nil,
//		//},
//		//{
//		//	[]clause.Interface{mclause.JsonBuild{
//		//		Fields: []clause.Column{clause.PrimaryColumn},
//		//	}, mclause.JsonBuild{
//		//		Fields: []clause.Column{{Name: "name"}},
//		//	}, clause.From{}},
//		//	"SELECT `name` FROM `users`", nil,
//		//},
//		//{
//		//	[]clause.Interface{mclause.JsonBuild{
//		//		Expression: clause.CommaExpression{
//		//			Exprs: []clause.Expression{
//		//				clause.NamedExpr{"?", []interface{}{clause.Column{Name: "id"}}},
//		//				clause.NamedExpr{"?", []interface{}{clause.Column{Name: "name"}}},
//		//				clause.NamedExpr{"LENGTH(?)", []interface{}{clause.Column{Name: "mobile"}}},
//		//			},
//		//		},
//		//	}, clause.From{}},
//		//	"SELECT `id`, `name`, LENGTH(`mobile`) FROM `users`", nil,
//		//},
//		//{
//		//	[]clause.Interface{mclause.JsonBuild{
//		//		Expression: clause.CommaExpression{
//		//			Exprs: []clause.Expression{
//		//				clause.Expr{
//		//					SQL: "? as name",
//		//					Vars: []interface{}{clause.Eq{
//		//						Column: clause.Column{Name: "age"},
//		//						Value:  18,
//		//					},
//		//					},
//		//				},
//		//			},
//		//		},
//		//	}, clause.From{}},
//		//	"SELECT `age` = ? as name FROM `users`", []interface{}{18},
//		//},
//	}
//
//	for idx, result := range results {
//		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
//			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
//		})
//	}
//}

var db, _ = gorm.Open(tests.DummyDialector{}, nil)

func checkBuildClauses(t *testing.T, clauses []clause.Interface, result string, vars []interface{}) {
	var (
		buildNames    []string
		buildNamesMap = map[string]bool{}
		user, _       = schema.Parse(&tests.User{}, &sync.Map{}, db.NamingStrategy)
		stmt          = gorm.Statement{DB: db, Table: user.Table, Schema: user, Clauses: map[string]clause.Clause{}}
	)

	for _, c := range clauses {
		if _, ok := buildNamesMap[c.Name()]; !ok {
			buildNames = append(buildNames, c.Name())
			buildNamesMap[c.Name()] = true
		}

		stmt.AddClause(c)
	}

	stmt.Build(buildNames...)

	testRes := strings.TrimSpace(stmt.SQL.String())

	if testRes != result {
		t.Errorf("SQL expects %v got %v", result, stmt.SQL.String())
	}

	//if !reflect.DeepEqual(stmt.Vars, vars) {
	//	t.Errorf("Vars expects %+v got %v", stmt.Vars, vars)
	//}
}
