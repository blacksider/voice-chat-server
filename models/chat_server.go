package models

type ChatServer struct {
	tableName   struct{} `pg:"chat_server"`
	Id          int64   `pg:"type:bigint,unique,notnull,pk"`
	Name        string   `pg:"type:varchar(255),notnull"`
	Description string   `pg:"type:varchar(255),notnull"`
}

type ChatServerData struct {
	Id          int64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
