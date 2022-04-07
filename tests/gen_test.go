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
	gName3 := "Group3"
	gName4 := "Group4"
	tName1 := "Tag1"
	tName2 := "Tag2"
	tName3 := "Tag3"

	iName1 := "Item1"
	iName2 := "Item2"
	iName3 := "Item3"
	iName4 := "Item4"
	iName5 := "Item5"
	iName6 := "Item6"
	iName7 := "Item7"
	iName8 := "Item8"

	ug1 := &models.UserGroup{Name: &gName1}
	ug2 := &models.UserGroup{Name: &gName2}
	ug3 := &models.UserGroup{Name: &gName3}
	ug4 := &models.UserGroup{Name: &gName4}

	iItem1 := &models.InnerItem{Name: &iName1}
	iItem2 := &models.InnerItem{Name: &iName2}
	iItem3 := &models.InnerItem{Name: &iName3}
	iItem4 := &models.InnerItem{Name: &iName4}
	iItem5 := &models.InnerItem{Name: &iName5}
	iItem6 := &models.InnerItem{Name: &iName6}
	iItem7 := &models.InnerItem{Name: &iName7}
	iItem8 := &models.InnerItem{Name: &iName8}

	item1 := &models.Item{Name: &iName1, Group: ug3, InnerItems: []*models.InnerItem{iItem5, iItem6}}
	item2 := &models.Item{Name: &iName2, Group: ug3, InnerItems: []*models.InnerItem{iItem6, iItem7}}
	item3 := &models.Item{Name: &iName3, Group: ug4, InnerItems: []*models.InnerItem{iItem7, iItem8}}
	item4 := &models.Item{Name: &iName4, Group: ug4, InnerItems: []*models.InnerItem{iItem8, iItem1}}

	tag1 := &models.Tag{Name: &tName1, InnerItems: []*models.InnerItem{iItem1, iItem2}, Items: []*models.Item{item1, item2}}
	tag2 := &models.Tag{Name: &tName2, InnerItems: []*models.InnerItem{iItem2, iItem3}, Items: []*models.Item{item2, item3}}
	tag3 := &models.Tag{Name: &tName3, InnerItems: []*models.InnerItem{iItem3, iItem4}, Items: []*models.Item{item3, item4}}

	db.Create(iItem1)
	db.Create(iItem2)
	db.Create(iItem3)
	db.Create(iItem4)

	db.Create(item1)
	db.Create(item2)
	db.Create(item3)
	db.Create(item4)

	db.Create(tag1)
	db.Create(tag2)
	db.Create(tag3)

	db.Create(ug1)
	db.Create(ug2)

	db.Create(&models.User{Name: &uName1, GroupID: ug1.ID, Tags: []*models.Tag{tag1, tag2}, Items: []*models.Item{item1, item2}})
	db.Create(&models.User{Name: &uName2, GroupID: ug2.ID, Tags: []*models.Tag{tag1, tag3}, Items: []*models.Item{item3, item4}})

	return db

}

func TestManyToManyRelation(t *testing.T) {
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'tags',\n(SELECT json_agg(json_build_object(\n'name',tags1_name)) FROM (  SELECT \"tags1\".\"name\" AS \"tags1_name\" FROM tags tags1 JOIN \"user_tag\" \"user_tag1\" ON (user_id = users0_id AND tag_id = id) ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\" FROM users users0 JOIN \"user_tag\" \"user_tag0\" ON \"users0\".\"id\" = \"user_tag0\".\"user_id\" JOIN \"tags\" \"Tags0\" ON \"user_tag0\".\"tag_id\" = \"Tags0\".\"id\" WHERE \"Tags0\".name = 'Tag3' ) as root"
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
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'group',\n(SELECT json_build_object(\n'name',user_groups1_name) FROM (  SELECT \"user_groups1\".\"name\" AS \"user_groups1_name\" FROM user_groups user_groups1 WHERE id = users0_group_id ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\",\"users0\".\"group_id\" AS \"users0_group_id\" FROM users users0 LEFT JOIN \"user_groups\" \"Group0\" ON \"users0\".\"group_id\" = \"Group0\".\"id\" WHERE \"Group0\".name = 'Group1' ) as root"
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
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'items',\n(SELECT json_agg(json_build_object(\n'name',items1_name)) FROM (  SELECT \"items1\".\"name\" AS \"items1_name\" FROM items items1 WHERE user_id = users0_id ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\" FROM users users0 LEFT JOIN \"items\" \"Items0\" ON \"users0\".\"id\" = \"Items0\".\"user_id\" WHERE \"Items0\".name = 'Item2' ) as root"
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

func TestManyToManyRelationWithManyToManyFieldFilter(t *testing.T) {
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'tags',\n(SELECT json_agg(json_build_object(\n'name',tags1_name)) FROM (  SELECT \"tags1\".\"name\" AS \"tags1_name\" FROM tags tags1 JOIN \"user_tag\" \"user_tag1\" ON (user_id = users0_id AND tag_id = id) JOIN \"tag_items\" \"tag_items1\" ON \"tags1\".\"id\" = \"tag_items1\".\"tag_id\" JOIN \"items\" \"Items1\" ON \"tag_items1\".\"item_id\" = \"Items1\".\"id\" WHERE \"Items1\".name = 'Item1' ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\" FROM users users0 ) as root"
	user := models.User{}
	db := Prepare()
	db.Callback().Query().Register("gorm:query", callbacks.Query)
	db = db.Session(&gorm.Session{DryRun: true})

	tagsQuery := db.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}}).Joins("Items").Joins("InnerItems").Where(db.Where("\"Items\".name = 'Item1'").Or("\"Items\".name = 'Item2'")).Or("\"InnerItems\".name = 'Item1'").Group("id").Order("name desc")
	stm := db.Table("users \"Users0\"").Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "tags", Query: tagsQuery, TargetType: &models.Tag{}},
		}}).Find(&user).Statement

	txt := stm.SQL.String()
	assert.Equal(t, txt, testSQL)
	log.Println(txt)
}

func TestManyToManyRelationWithInnerManyToManyFieldFilter(t *testing.T) {
	testSQL := "SELECT json_build_object(\n'id',users0_id,\n'name',users0_name,\n'tags',\n(SELECT json_agg(json_build_object(\n'name',tags1_name)) FROM (  SELECT \"tags1\".\"name\" AS \"tags1_name\" FROM tags tags1 JOIN \"user_tag\" \"user_tag1\" ON (user_id = users0_id AND tag_id = id) JOIN \"tag_items\" \"tag_items1\" ON \"tags1\".\"id\" = \"tag_items1\".\"tag_id\" JOIN \"items\" \"Items1\" ON \"tag_items1\".\"item_id\" = \"Items1\".\"id\" WHERE \"Items1\".name = 'Item1' ) as root)\n) FROM (  SELECT \"users0\".\"id\" AS \"users0_id\",\"users0\".\"name\" AS \"users0_name\" FROM users users0 ) as root"
	user := models.User{}
	db := Prepare()
	db.Callback().Query().Register("gorm:query", callbacks.Query)
	db = db.Session(&gorm.Session{DryRun: true})

	tagsQuery := db.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}}).Joins("Items").Joins("Items.InnerItems").Joins("Items.Group").Where(db.Where("\"Items\".name = 'Item1'").Or("\"Items\".name = 'Item2'")).Or(db.Where("\"Items.InnerItems\".name = 'Item5'").Where("\"Items.Group\".name = 'Group3'")).Group("id").Limit(10)
	stm := db.Table("users \"Users0\"").Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "tags", Query: tagsQuery, TargetType: &models.Tag{}},
		}}).Find(&user).Statement

	txt := stm.SQL.String()
	assert.Equal(t, txt, testSQL)
	log.Println(txt)
}
