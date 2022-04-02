package tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"sqlgenerator/callbacks"
	"sqlgenerator/mclause"
	"sqlgenerator/models"
	"testing"
)

func Truncate(db *gorm.DB) error {
	tx := db.Begin()
	var tableNames []string
	q := "SELECT table_name FROM information_schema.tables where table_schema = 'public';"
	tx.Raw(q).Scan(&tableNames)
	if tx.Error != nil {
		tx.Rollback()
		return tx.Error
	}
	for _, t := range tableNames {
		err := tx.Exec(fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE;", t)).Error
		if err != nil {
			tx.Rollback()
			return tx.Error
		}
	}
	return tx.Commit().Error
}

func Prepare() *gorm.DB {
	dsn := "host=localhost user=postgres password=postgres dbname=gen_test port=5432 sslmode=disable TimeZone=Europe/Moscow"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(models.User{})
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(models.UserGroup{})
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(models.Tag{})
	if err != nil {
		panic(err)
	}

	err = Truncate(db)
	if err != nil {
		panic(err)
	}

	uName1 := "Vasya"
	uName2 := "Petya"

	tName1 := "Tag1"
	tName2 := "Tag2"
	tName3 := "Tag3"

	tag1 := &models.Tag{Name: &tName1}
	tag2 := &models.Tag{Name: &tName2}
	tag3 := &models.Tag{Name: &tName3}

	ug1 := &models.UserGroup{Name: &uName1}
	ug2 := &models.UserGroup{Name: &uName2}

	db.Create(tag1)
	db.Create(tag2)
	db.Create(tag3)

	db.Create(ug1)
	db.Create(ug2)

	db.Create(&models.User{Name: &uName1, GroupID: ug1.ID, Tags: []*models.Tag{tag1, tag2}})
	db.Create(&models.User{Name: &uName2, GroupID: ug2.ID, Tags: []*models.Tag{tag1, tag3}})

	return db

}

//func TestGenerateCode(t *testing.T) {
//	testSQL := "SELECT json_build_object('id',users.id,'name',users.name) FROM (SELECT users.name,users.id,\"Group\".\"id\",\"Group\".\"name\" FROM \"users\" LEFT JOIN \"user_groups\" \"Group\" ON \"users\".\"group_id\" = \"Group\".\"id\" WHERE \"Group\".name = $1) as root ORDER BY \"root\".\"id\" LIMIT 1"
//	user := models.User{}
//	db := Prepare()
//	db.Callback().Query().Register("gorm:query", callbacks.Query)
//	subQuery1 := db.Model(&user).Joins("Group").Select("users.name,users.id,\"Group\".\"id\",\"Group\".\"name\"").Where("\"Group\".name = ?", "vasya")
//	stm := db.Session(&gorm.Session{DryRun: true}).Table("(?) as root", subQuery1).Clauses(mclause.JsonBuild{Fields: []clause.Column{{Name: "id", Alias: "users.id"}, {Name: "name", Alias: "users.name"}}}).First(&user).Statement
//	txt := stm.SQL.String()
//	assert.Equal(t, txt, testSQL)
//	log.Println(txt)
//}

func TestManyToManyRelation(t *testing.T) {
	testSQL := "SELECT json_build_object('id',users_id,'name',users_name) FROM (SELECT \"users\".\"name\" AS \"users_name\",\"users\".\"id\" AS \"users_id\" FROM \"users\" LEFT JOIN \"user_tag\" \"user_tag\" ON \"users\".\"id\" = \"user_tag\".\"user_id\" LEFT JOIN \"tags\" \"Tags\" ON \"user_tag\".\"tag_id\" = \"Tags\".\"id\" WHERE \"Tags\".name = 'Tag1') as root"
	user := models.User{}
	db := Prepare()
	db.Callback().Query().Register("gorm:query", callbacks.Query)
	db = db.Session(&gorm.Session{DryRun: true})

	tagsQuery := db.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
		}})

	stm := db.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "tags", Query: tagsQuery, TargetType: &models.Tag{}},
		}}).Joins("Tags").Where("\"Tags\".name = 'Tag1'").Find(&user).Statement

	//
	//subQueryL2 := db.Model(&models.Tag{}).Clauses(mclause.JsonQueryField{Fields: []mclause.Column{
	//	{Name: "tags.name", Path: "tags1_name"},
	//	{Name: "tags.id", Path: "tags1_id"},
	//}})
	//
	//db = db.Session(&gorm.Session{DryRun: true})
	//
	//stmL2 := db.Table("(?) as root", subQueryL2).Clauses(mclause.JsonBuild{
	//	JsonAgg: true,
	//	Fields: []mclause.Column{
	//		{Name: "id", Path: "tags1_id"},
	//		{Name: "name", Path: "tags1_name"},
	//	},
	//})
	//
	//stm := db.Table("(?) as root", subQuery).Clauses(mclause.JsonBuild{
	//	Fields: []mclause.Column{
	//		{Name: "id", Path: "users0_id"},
	//		{Name: "name", Path: "users0_name"},
	//		{Name: "tags", Query: db.Table("(?) as root", stmL2).Find(models.Tag{})},
	//	},
	//}).Find(&user).Statement

	txt := stm.SQL.String()
	assert.Equal(t, txt, testSQL)
	log.Println(txt)
}
