package tests

import (
	"fmt"
	"github.com/mirogindev/maker_sql/callbacks"
	"github.com/mirogindev/maker_sql/mclause"
	"github.com/mirogindev/maker_sql/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

var DB *gorm.DB

func init() {
	dsn := "host=localhost user=postgres password=postgres dbname=gen_test port=5432 sslmode=disable TimeZone=Europe/Moscow"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	err = DB.AutoMigrate(models.Status{})
	if err != nil {
		panic(err)
	}

	err = DB.AutoMigrate(models.User{})
	if err != nil {
		panic(err)
	}
	err = DB.AutoMigrate(models.UserGroup{})
	if err != nil {
		panic(err)
	}

	err = DB.AutoMigrate(models.Tag{})
	if err != nil {
		panic(err)
	}

	err = DB.AutoMigrate(models.Item{})
	if err != nil {
		panic(err)
	}

}

func Prepare() {

	err := Truncate(DB)
	DB.Callback().Query().Replace("gorm:query", callbacks.Query)
	if err != nil {
		panic(err)
	}

	status1 := "Status1"
	status2 := "Status2"
	status3 := "Status3"
	status4 := "Status4"

	uName1 := "User1"
	uName2 := "User2"

	gName1 := "Group1"
	gName2 := "Group2"
	gName3 := "Group3"
	gName4 := "Group4"

	tName1 := "Tag1"
	tName2 := "Tag2"
	tName3 := "Tag3"

	tAggName1 := "aGroup1"
	tAggName2 := "aGroup2"
	//tAggName3 := "aGroup3"

	tAggVal1 := 20
	tAggVal2 := 30
	tAggVal3 := 15

	iName1 := "Item1"
	iName2 := "Item2"
	iName3 := "Item3"
	iName4 := "Item4"
	iName5 := "Item5"
	iName6 := "Item6"
	iName7 := "Item7"
	iName8 := "Item8"

	st1 := &models.Status{Name: &status1}
	st2 := &models.Status{Name: &status2}
	st3 := &models.Status{Name: &status3}
	st4 := &models.Status{Name: &status4}

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

	item1 := &models.Item{Name: &iName1, Group: ug3, Statuses: []*models.Status{st4}, InnerItems: []*models.InnerItem{iItem5, iItem6}}
	item2 := &models.Item{Name: &iName2, Group: ug3, Statuses: []*models.Status{st3}, InnerItems: []*models.InnerItem{iItem6, iItem7}}
	item3 := &models.Item{Name: &iName3, Group: ug4, Statuses: []*models.Status{st2}, InnerItems: []*models.InnerItem{iItem7, iItem8}}
	item4 := &models.Item{Name: &iName4, Group: ug4, Statuses: []*models.Status{st1}, InnerItems: []*models.InnerItem{iItem8, iItem1}}

	tag1 := &models.Tag{Name: &tName1, AggrName: &tAggName1, AggrVal: &tAggVal1, InnerItems: []*models.InnerItem{iItem1, iItem2}, Items: []*models.Item{item1, item2}}
	tag2 := &models.Tag{Name: &tName2, AggrName: &tAggName2, AggrVal: &tAggVal2, InnerItems: []*models.InnerItem{iItem2, iItem3}, Items: []*models.Item{item2, item3}}
	tag3 := &models.Tag{Name: &tName3, AggrName: &tAggName2, AggrVal: &tAggVal3, InnerItems: []*models.InnerItem{iItem3, iItem4}, Items: []*models.Item{item3, item4}}

	DB.Create(st1)
	DB.Create(st2)
	DB.Create(st3)
	DB.Create(st4)

	DB.Create(iItem1)
	DB.Create(iItem2)
	DB.Create(iItem3)
	DB.Create(iItem4)

	DB.Create(item1)
	DB.Create(item2)
	DB.Create(item3)
	DB.Create(item4)

	DB.Create(tag1)
	DB.Create(tag2)
	DB.Create(tag3)

	DB.Create(ug1)
	DB.Create(ug2)

	DB.Create(&models.User{Name: &uName1, AggrVal: 20, GroupID: ug1.ID, Tags: []*models.Tag{tag1, tag2}, Items: []*models.Item{item1, item2}})
	DB.Create(&models.User{Name: &uName2, AggrVal: 30, GroupID: ug2.ID, Tags: []*models.Tag{tag1, tag3}, Items: []*models.Item{item3, item4}})

}

func TestSimpleQuery(t *testing.T) {
	var users []*models.User
	Prepare()

	err := DB.Debug().Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
		}}).Find(&users).Error
	if err != nil {
		panic(err)
	}
	assert.NotEmpty(t, users)
	assert.Len(t, users, 2)
}

func TestManyToManyRelation(t *testing.T) {
	var users []*models.User
	Prepare()

	tagsQuery := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}})

	err := DB.Debug().Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "tags", Query: tagsQuery, TargetType: &models.Tag{}},
		}}).Joins("Tags").Where("\"Tags\".name = ?", "Tag3").Find(&users).Error
	if err != nil {
		panic(err)
	}
	assert.NotEmpty(t, users)
	assert.Len(t, users, 1)
	assert.Len(t, users[0].Tags, 2)
}

func TestManyToOneRelation(t *testing.T) {
	var users []*models.User
	Prepare()

	userGroupQuery := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}})

	err := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "group", Query: userGroupQuery, TargetType: &models.UserGroup{}},
		}}).Joins("Group").Where("\"Group\".name = ?", "Group1").Find(&users).Error

	if err != nil {
		panic(err)
	}

	assert.Len(t, users, 1)
	assert.NotEmpty(t, users[0].Group)
}

func TestOneToManyRelation(t *testing.T) {
	var users []*models.User
	Prepare()
	itemsQuery := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}})

	err := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "items", Query: itemsQuery, TargetType: &models.Item{}},
		}}).Joins("Items").Where("\"Items\".name = ?", "Item2").Find(&users).Error

	if err != nil {
		panic(err)
	}

	assert.Len(t, users, 1)
	assert.Len(t, users[0].Items, 2)

}

func TestManyToManyRelationWithManyToManyFieldFilter(t *testing.T) {
	var users []*models.User
	Prepare()

	tagsQuery := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}}).Joins("Items").Joins("InnerItems").Where(DB.Where("\"Items\".name = ?", "Item1").Or("\"Items\".name = ?", "Item2")).Or("\"InnerItems\".name = ?", "Item1").Group("id").Order("name desc")
	err := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "tags", Query: tagsQuery, TargetType: &models.Tag{}},
		}}).Find(&users).Error

	if err != nil {
		panic(err)
	}

	assert.Len(t, users, 2)
	assert.Len(t, users[0].Tags, 2)
	assert.Len(t, users[1].Tags, 1)

}

func TestManyToManyRelationWithInnerManyToManyFieldFilter(t *testing.T) {
	var users []*models.User
	Prepare()

	tagsQuery := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "name"},
		}}).Joins("Items").Joins("Items.InnerItems").Joins("Items.Statuses").Joins("Items.Group").Where(DB.Where("\"Items\".name = 'Item1'").Or("\"Items\".name = ?", "Item2")).Or(db.Where("\"Items.InnerItems\".name = ?", "Item7").Where("\"Items.Group\".name = ?", "Group3").Where("\"Items.Statuses\".name = ?", "Status1")).Group("id").Limit(10)

	err := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "name"},
			{Name: "tags", Query: tagsQuery, TargetType: &models.Tag{}},
		}}).Find(&users).Error

	if err != nil {
		panic(err)
	}

	assert.Len(t, users, 2)
	assert.Len(t, users[0].Tags, 2)
	assert.Len(t, users[1].Tags, 1)
}

func TestSumAggregation(t *testing.T) {
	var users []*models.UserAggregate
	Prepare()

	err := DB.Model(&models.User{}).Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "sum", AggrQuery: &mclause.AggrQuery{Type: mclause.Sum, Fields: []string{"aggr_val"}}},
		}}).Find(&users).Error

	if err != nil {
		panic(err)
	}

	assert.Len(t, users, 2)
	assert.Equal(t, *users[0].Sum.AggrVal, 30)
	assert.Equal(t, *users[1].Sum.AggrVal, 20)
}

func TestInnerSumAggregation(t *testing.T) {
	var users []*models.User
	Prepare()

	tagsAggQuery := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "aggr_name"},
			{Name: "sum", AggrQuery: &mclause.AggrQuery{Type: mclause.Sum, Fields: []string{"aggr_val"}}},
		}})

	err := DB.Clauses(mclause.JsonBuild{
		Fields: []mclause.Field{
			{Name: "id"},
			{Name: "tags_aggregate", Query: tagsAggQuery, TargetType: &models.Tag{}}},
	}).Find(&users).Error

	if err != nil {
		panic(err)
	}

	assert.Len(t, users, 2)
	assert.Len(t, users[0].TagsAggregate, 2)
	assert.Len(t, users[1].TagsAggregate, 2)
	assert.Equal(t, *users[0].TagsAggregate[0].Sum.AggrVal, 20)
	assert.Equal(t, *users[0].TagsAggregate[1].Sum.AggrVal, 30)
	assert.Equal(t, *users[1].TagsAggregate[0].Sum.AggrVal, 20)
	assert.Equal(t, *users[1].TagsAggregate[1].Sum.AggrVal, 15)
}
