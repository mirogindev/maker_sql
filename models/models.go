package models

import "time"

type User struct {
	ID        *int64     `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	CreatedAt time.Time  `json:"created_at" mapstructure:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" mapstructure:"updated_at"`
	Name      *string    `json:"name" mapstructure:"name"`
	Tags      []*Tag     `json:"tags" gorm:"many2many:user_tag"`
	GroupID   *int64     `json:"group_id" mapstructure:"group_id" graphql:"group_id"`
	Group     *UserGroup `json:"group" mapstructure:"group" gorm:"foreignKey:GroupID"`
}

type Tag struct {
	ID    *int64  `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	Name  *string `json:"name"`
	Users []*User `json:"users"  gorm:"many2many:user_tag"`
}

type UserGroup struct {
	ID        *int64    `json:"id" mapstructure:"id" gorm:"primarykey" graphql:"id"`
	CreatedAt time.Time `json:"created_at" mapstructure:"created_at"`
	UpdatedAt time.Time `json:"updated_at" mapstructure:"updated_at"`
	Name      *string   `json:"name" mapstructure:"name" graphql:"name"`
	Users     []*User   `json:"users" gorm:"foreignKey:GroupID"  table:"users" foreignKeyName:"group_id"`
}
