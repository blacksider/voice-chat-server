package service

import (
	"github.com/go-pg/pg/v9/orm"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"voice-chat-server/logger"
	"voice-chat-server/models"
)

type ChatUserService struct {
	DbService *DBService
}

func (service *ChatUserService) Init() error {
	logger.Logger.Info("Init ChatUserService")
	for _, model := range []interface{}{(*models.ChatUser)(nil)} {
		err := service.DbService.DB.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists:   true,
			FKConstraints: true,
		})
		if err != nil {
			return err
		}
	}

	user := service.GetUserByUsername("admin")
	if user == nil {
		logger.Logger.Info("Init default user: 'admin'")
		hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		encodePW := string(hash)
		err = service.DbService.DB.Insert(&models.ChatUser{
			Id:       1,
			Name:     "admin",
			UserName: "admin",
			Password: encodePW,
		})
		if err != nil {
			return err
		}
		logger.Logger.Info("Init default user succeed")
	}
	return nil
}

func (service *ChatUserService) GetUserByUsername(username string) *models.ChatUser {
	var user models.ChatUser
	err := service.DbService.DB.Model(&user).Where("user_name = ?", username).Select()
	if err != nil {
		logger.Logger.Error(err)
		return nil
	}
	return &user
}

func (service *ChatUserService) UpdatePassword(username string, newPwd string) *models.ChatUser {
	user := service.GetUserByUsername(username)

	if user == nil {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return nil
	}
	encodePW := string(hash)
	user.Password = encodePW

	err = service.DbService.DB.Update(&user)
	if err != nil {
		return nil
	}
	return user
}

func (service *ChatUserService) AuthUser(username string, password string) *models.ChatUser {
	usernameQL := service.GetUserByUsername(username)

	if usernameQL == nil {
		return nil
	}
	err := bcrypt.CompareHashAndPassword([]byte(usernameQL.Password), []byte(password))
	if err != nil {
		logger.Logger.Error(err)
		return nil
	}

	return usernameQL
}
