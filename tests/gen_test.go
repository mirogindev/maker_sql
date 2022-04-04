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

	err = db.AutoMigrate(models.Item{})
	if err != nil {
		panic(err)
	}

	err = Truncate(db)
	if err != nil {
		panic(err)
	}

	uName1 := "User1"
	uName2 := "User2"

	gName1 := "Group1"
	gName2 := "Group2"

	tName1 := "Tag1"
	tName2 := "Tag2"
	tName3 := "Tag3"

	iName1 := "Item1"
	iName2 := "Item2"
	iName3 := "Item3"
	iName4 := "Item4"

	tag1 := &models.Tag{Name: &tName1}
	tag2 := &models.Tag{Name: &tName2}
	tag3 := &models.Tag{Name: &tName3}

	ug1 := &models.UserGroup{Name: &gName1}
	ug2 := &models.UserGroup{Name: &gName2}

	item1 := &models.Item{Name: &iName1}
	item2 := &models.Item{Name: &iName2}
	item3 := &models.Item{Name: &iName3}
	item4 := &models.Item{Name: &iName4}

	db.Create(tag1)
	db.Create(tag2)
	db.Create(tag3)

	db.Create(item1)
	db.Create(item2)
	db.Create(item3)
	db.Create(item4)

	db.Create(ug1)
	db.Create(ug2)

	db.Create(&models.User{Name: &uName1, GroupID: ug1.ID, Tags: []*models.Tag{tag1, tag2}, Items: []*models.Item{item1, item2}})
	db.Create(&models.User{Name: &uName2, GroupID: ug2.ID, Tags: []*models.Tag{tag1, tag3}, Items: []*models.Item{item3, item4}})

	return db

}

func TestManyToManyRelation(t *testing.T) {
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'tags',\n(SELECT json_agg(json_build_object(\n'name',tags1_name)) FROM (  SELECT \"tags1\".\"name\" AS \"tags1_name\" FROM tags tags1 JOIN \"user_tag\" \"user_tag1\" ON (tag_id = id AND user_id = users0_id) ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\" FROM users users0 JOIN \"user_tag\" \"user_tag\" ON \"users0\".\"id\" = \"user_tag\".\"user_id\" JOIN \"tags\" \"Tags\" ON \"user_tag\".\"tag_id\" = \"Tags\".\"id\" WHERE \"Tags\".name = 'Tag3' ) as root"
	user := models.User{}
	db := Prepare()
	db.Callback().Query().Register("gorm:query", callbacks.Query)
	db = db.Session(&gorm.Session{DryRun: true})

	tagsQuery := db.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}})

	stm := db.Table("users users0").Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "tags", Query: tagsQuery, TargetType: &models.Tag{}},
		}}).Joins("Tags").Where("\"Tags\".name = 'Tag3'").Find(&user).Statement

	txt := stm.SQL.String()
	assert.Equal(t, txt, testSQL)
	log.Println(txt)
}

func TestManyToOneRelation(t *testing.T) {
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'group',\n(SELECT json_build_object(\n'name',user_groups1_name) FROM (  SELECT \"user_groups1\".\"name\" AS \"user_groups1_name\" FROM user_groups user_groups1 WHERE id = users0_group_id ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\",\"users0\".\"group_id\" AS \"users0_group_id\" FROM users users0 LEFT JOIN \"user_groups\" \"Group\" ON \"users0\".\"group_id\" = \"Group\".\"id\" WHERE \"Group\".name = 'Group1' ) as root"
	user := models.User{}
	db := Prepare()
	db.Callback().Query().Register("gorm:query", callbacks.Query)
	db = db.Session(&gorm.Session{DryRun: true})

	userGroupQuery := db.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}})

	stm := db.Table("users users0").Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "group", Query: userGroupQuery, TargetType: &models.UserGroup{}},
		}}).Joins("Group").Where("\"Group\".name = 'Group1'").Find(&user).Statement

	txt := stm.SQL.String()
	assert.Equal(t, txt, testSQL)
	log.Println(txt)
}

func TestOneToManyRelation(t *testing.T) {
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'items',\n(SELECT json_agg(json_build_object(\n'name',items1_name)) FROM (  SELECT \"items1\".\"name\" AS \"items1_name\" FROM items items1 WHERE user_id = users0_id ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\" FROM users users0 LEFT JOIN \"items\" \"Items\" ON \"users0\".\"id\" = \"Items\".\"user_id\" WHERE \"Items\".name = 'Item2' ) as root"
	user := models.User{}
	db := Prepare()
	db.Callback().Query().Register("gorm:query", callbacks.Query)
	db = db.Session(&gorm.Session{DryRun: true})

	itemsQuery := db.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}})

	stm := db.Table("users users0").Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "items", Query: itemsQuery, TargetType: &models.Item{}},
		}}).Joins("Items").Where("\"Items\".name = 'Item2'").Find(&user).Statement

	txt := stm.SQL.String()
	assert.Equal(t, txt, testSQL)
	log.Println(txt)
}
