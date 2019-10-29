package models

type ChatUser struct {
	tableName struct{} `pg:"chat_user"`
	Id        int64    `json:"id" pg:"type:bigint,unique,notnull,pk"`
	Name      string   `json:"name" pg:"type:varchar(255),notnull"`
	UserName  string   `json:"username" pg:"type:varchar(255),unique,notnull"`
	Password  string   `json:"password" pg:"type:varchar(255),notnull"`
}
