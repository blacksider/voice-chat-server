package service

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"net/http"
	"time"
	"voice-chat-server/logger"
	"voice-chat-server/models"
	"voice-chat-server/utils/auth"
)

const DefaultExpiration = int64(30 * time.Minute / time.Millisecond)

type SessionService struct {
	DbService   *DBService
	UserService *ChatUserService
	ticker      *time.Ticker
}

func (service *SessionService) Init() error {
	logger.Logger.Info("Init SessionService")
	for _, model := range []interface{}{(*models.UserSession)(nil)} {
		err := service.DbService.DB.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists:   true,
			FKConstraints: true,
		})
		if err != nil {
			return err
		}
	}
	service.startCheckSessionScheduler()
	return nil
}

func (service *SessionService) startCheckSessionScheduler() {
	logger.Logger.Info("Start scheduler of checking session expiration")
	service.ticker = time.NewTicker(30 * time.Second)
	go func() {
		for t := range service.ticker.C {
			_ = t
			service.checkSessions()
		}
	}()
}

func (service *SessionService) checkSessions() {
	logger.Logger.Info("Checking sessions' expiration")
	session := models.UserSession{}
	now := time.Now().UnixNano() / int64(time.Millisecond)
	r, err := service.DbService.DB.Model(&session).
		Where("expires < ?", now).
		Delete()
	if err != nil {
		logger.Logger.Error(err)
	}
	logger.Logger.Info("Expired total", r.RowsAffected())
}

func (service *SessionService) Close() {
	service.ticker.Stop()
	logger.Logger.Info("Scheduler of checking session expiration closed")
}

func (service *SessionService) Create(user *models.ChatUser, token string) *models.UserSession {
	var session models.UserSession
	err := service.DbService.DB.RunInTransaction(func(tx *pg.Tx) error {
		oldSession := service.GetByUserName(user.UserName)
		if oldSession != nil {
			err := service.DbService.DB.Delete(oldSession)
			if err != nil {
				logger.Logger.Error(err)
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
			logger.Logger.Error(err)
			return err
		}
		return nil
	})

	if err != nil {
		logger.Logger.Error(err)
		return nil
	}
	return &session
}

func (service *SessionService) GetByUserName(user string) *models.UserSession {
	var session models.UserSession
	err := service.DbService.DB.Model(&session).Where("user_name = ?", user).Select()
	if err != nil {
		logger.Logger.Error(err)
		return nil
	}
	return &session
}

func (service *SessionService) GetByToken(token string) *models.UserSession {
	var session models.UserSession
	err := service.DbService.DB.Model(&session).Where("token = ?", token).Select()
	if err != nil {
		logger.Logger.Error(err)
		return nil
	}
	return &session
}

func (service *SessionService) GetUserFromRequest(w http.ResponseWriter, r *http.Request) *models.ChatUser {
	jwtToken := auth.GetTokenFromRequest(r)
	return service.GetUserByJwtToken(jwtToken, w)
}

func (service *SessionService) GetUserFromRequestParam(w http.ResponseWriter, r *http.Request) *models.ChatUser {
	jwtToken := auth.GetTokenFromRequestParam(r)
	return service.GetUserByJwtToken(jwtToken, w)
}

func (service *SessionService) GetUserByJwtToken(jwtToken *jwt.Token, w http.ResponseWriter) *models.ChatUser {
	if jwtToken == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "Token not found")
		return nil
	}
	logger.Logger.Debug("Found token:", jwtToken.Raw)
	session := service.GetByToken(jwtToken.Raw)
	if session == nil || session.IsExpired() {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "Token expired")
		return nil
	}
	user := service.UserService.GetUserByUsername(session.UserName)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "User not found")
		return nil
	}
	return user
}
