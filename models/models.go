package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (j *User) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}
	var result User

	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j User) Value() (driver.Value, error) {
	res, err := json.Marshal(j)
	return res, err
}

func (j *UserAggregate) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}
	var result UserAggregate

	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j UserAggregate) Value() (driver.Value, error) {
	res, err := json.Marshal(j)
	return res, err
}

type UserAggregate struct {
	ID  *int64            `json:"id"`
	Sum *UserAggregateSum `json:"sum" gorm:"-"`
}

type UserAggregateSum struct {
	AggrVal  *int `json:"aggr_val"`
	AggrVal2 *int `json:"aggr_val2"`
}

type TagAggregateSum struct {
	AggrVal *int `json:"aggr_val"`
}

//type BudgetMonthReservedAggregate struct {
//	ID   *int64            `mapstructure:"id" graphql:"id" json:"id"`
//	Name *string           `json:"name" mapstructure:"name"`
//	Sum  *UserAggregateSum `json:"sum" mapstructure:"sum" graphql:"sum"`
//}

type User struct {
	ID            *int64          `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	CreatedAt     time.Time       `json:"created_at" mapstructure:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" mapstructure:"updated_at"`
	Name          *string         `json:"name" mapstructure:"name"`
	Tags          []*Tag          `json:"tags" gorm:"many2many:user_tag"`
	TagsAggregate []*TagAggregate `json:"tags_aggregate" gorm:"-" sql_gen:"Tags"`
	Items         []*Item         `json:"items" gorm:"foreignKey:UserId"`
	AggrVal       int             `json:"aggr_val"`
	AggrVal2      int             `json:"aggr_val2"`
	GroupID       *int64          `json:"group_id" mapstructure:"group_id" graphql:"group_id"`
	Group         *UserGroup      `json:"group" mapstructure:"group" gorm:"foreignKey:GroupID"`
	IsAdmin       *bool           `json:"is_admin" mapstructure:"is_admin" graphql:"is_admin"`
}

type Status struct {
	ID     *int64  `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	ItemID *int64  `json:"item_id"`
	Name   *string `json:"name"`
}

type Item struct {
	ID         *int64       `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	Name       *string      `json:"name"`
	InnerItems []*InnerItem `gorm:"many2many:item_inner_items"`
	GroupID    *int64       `json:"group_id" mapstructure:"group_id" graphql:"group_id"`
	UserId     *int64       `json:"user_id"`
	Group      *UserGroup   `json:"group" mapstructure:"group" gorm:"foreignKey:GroupID"`
	Statuses   []*Status    `json:"statuses" mapstructure:"statuses" gorm:"foreignKey:ItemID"`
}

type InnerItem struct {
	ID   *int64  `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	Name *string `json:"name"`
}

type Tag struct {
	ID         *int64       `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	Name       *string      `json:"name"`
	AggrName   *string      `json:"aggr_name"`
	AggrVal    *int         `json:"aggr_val"`
	AggrVal2   *int         `json:"aggr_vav2"`
	Users      []*User      `json:"users"  gorm:"many2many:user_tag"`
	Items      []*Item      `json:"items"  gorm:"many2many:tag_items"`
	InnerItems []*InnerItem `gorm:"many2many:tag_inner_items"`
}

type TagAggregate struct {
	ID       *int64           `json:"id"`
	AggrName *string          `json:"aggr_name"`
	Sum      *TagAggregateSum `json:"sum"`
}

type UserGroup struct {
	ID           *int64    `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	CreatedAt    time.Time `json:"created_at" mapstructure:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" mapstructure:"updated_at"`
	Name         *string   `json:"name" mapstructure:"name" graphql:"name"`
	Users        []*User   `json:"users" gorm:"foreignKey:GroupID"  table:"users" foreignKeyName:"group_id"`
	StatusID     *int64    `json:"group_id" mapstructure:"group_id" graphql:"group_id"`
	Status       *Status   `json:"status" mapstructure:"status" gorm:"foreignKey:StatusID"`
	StatusesMany []*Status `json:"statuses_many" mapstructure:"statuses_many" gorm:"many2many:usergroup_statuses"`
}
