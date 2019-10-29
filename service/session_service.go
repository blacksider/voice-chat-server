package service

import (
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"log"
	"time"
	"voice-chat-server/models"
)

const DefaultExpiration = int64(30 * time.Minute)

type SessionService struct {
	DbService *DBService
}

func (service *SessionService) Init() error {
	for _, model := range []interface{}{(*models.UserSession)(nil)} {
		err := service.DbService.DB.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists:   true,
			FKConstraints: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (service *SessionService) Create(user *models.ChatUser, token string) *models.UserSession {
	var session models.UserSession
	err := service.DbService.DB.RunInTransaction(func(tx *pg.Tx) error {
		oldSession := service.GetByUserName(user.UserName)
		if oldSession != nil {
			err := service.DbService.DB.Delete(oldSession)
			if err != nil {
				log.Println(err)
				return err
			}
		}

		now := time.Now().UnixNano() / int64(time.Millisecond)
		session = models.UserSession{
			UserName: user.UserName,
			Token:    token,
			CreateAt: now,
			Expires:  now + DefaultExpiration,
		}

		err := service.DbService.DB.Insert(&session)
		if err != nil {
			log.Println(err)
			return err
		}
		return nil
	})

	if err != nil {
		log.Println(err)
		return nil
	}
	return &session
}

func (service *SessionService) GetByUserName(user string) *models.UserSession {
	var session models.UserSession
	err := service.DbService.DB.Model(&session).Where("user_name = ?", user).Select()
	if err != nil {
		log.Println(err)
		return nil
	}
	return &session
}

func (service *SessionService) GetByToken(token string) *models.UserSession {
	var session models.UserSession
	err := service.DbService.DB.Model(&session).Where("token = ?", token).Select()
	if err != nil {
		log.Println(err)
		return nil
	}
	return &session
}
