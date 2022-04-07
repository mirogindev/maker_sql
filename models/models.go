package models

import "time"

type User struct {
	ID        *int64     `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	CreatedAt time.Time  `json:"created_at" mapstructure:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" mapstructure:"updated_at"`
	Name      *string    `json:"name" mapstructure:"name"`
	Tags      []*Tag     `json:"tags" gorm:"many2many:user_tag"`
	Items     []*Item    `json:"items" gorm:"foreignKey:UserId"`
	GroupID   *int64     `json:"group_id" mapstructure:"group_id" graphql:"group_id"`
	Group     *UserGroup `json:"group" mapstructure:"group" gorm:"foreignKey:GroupID"`
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
	Users      []*User      `json:"users"  gorm:"many2many:user_tag"`
	Items      []*Item      `json:"items"  gorm:"many2many:tag_items"`
	InnerItems []*InnerItem `gorm:"many2many:tag_inner_items"`
}

type UserGroup struct {
	ID        *int64    `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	CreatedAt time.Time `json:"created_at" mapstructure:"created_at"`
	UpdatedAt time.Time `json:"updated_at" mapstructure:"updated_at"`
	Name      *string   `json:"name" mapstructure:"name" graphql:"name"`
	Users     []*User   `json:"users" gorm:"foreignKey:GroupID"  table:"users" foreignKeyName:"group_id"`
}
