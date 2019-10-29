package models

type ChatRoom struct {
	tableName   struct{} `pg:"chat_room"`
	Id          int64    `json:"id" pg:"type:bigint,unique,notnull,pk"`
	Name        string   `json:"name" pg:"type:varchar(255),notnull"`
	Description string   `json:"description" pg:"type:varchar(255),notnull"`
	ServerId    int64   `pg:"on_delete:RESTRICT, on_update: CASCADE"`
	Server      *ChatServer
}
