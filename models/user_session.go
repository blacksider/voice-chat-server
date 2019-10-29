package models

import "time"

type UserSession struct {
	tableName struct{} `pg:"chat_user_session"`
	UserName  string   `pg:"type:varchar(255),unique,notnull"`
	Token     string   `pg:"type:varchar(255),unique,notnull"`
	CreateAt  int64    `pg:"type:bigint,notnull"`
	Expires   int64    `pg:"type:bigint,notnull"`
}

func (session *UserSession) IsExpired() bool {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	return session.Expires < now
}
