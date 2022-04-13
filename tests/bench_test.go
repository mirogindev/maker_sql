package tests

import (
	"github.com/mirogindev/maker_sql/mclause"
	"github.com/mirogindev/maker_sql/models"
	"gorm.io/gorm"
	"testing"
)

func BenchmarkSQLGeneration(b *testing.B) {
	var users []*models.User
	Prepare()

	for i := 0; i < b.N; i++ {
		tagsQuery := DB.Session(&gorm.Session{DryRun: true}).Clauses(mclause.JsonBuild{
			Fields: []mclause.Field{
				{Name: "name"},
			}}).Joins("Items").Joins("InnerItems").Where(DB.Where("\"Items\".name = ?", "Item1").Or("\"Items\".name = ?", "Item2")).Or("\"InnerItems\".name = ?", "Item1").Group("id").Order("name desc")

		DB.Session(&gorm.Session{DryRun: true}).Clauses(mclause.JsonBuild{
			Fields: []mclause.Field{
				{Name: "id"},
				{Name: "name"},
				{Name: "tags", Query: tagsQuery},
			}}).Find(&users).Statement.SQL.String()
	}
}
