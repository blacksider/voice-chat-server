package models

type ChatUserConnStats struct {
	Id     string `json:"id" pg:"type:varchar(255),unique,notnull,pk"`
	UserId int64  `pg:"on_delete:RESTRICT, on_update: CASCADE"`
	User   *ChatUser
	RoomId int64 `pg:"on_delete:RESTRICT, on_update: CASCADE"`
	Room   *ChatRoom
}
