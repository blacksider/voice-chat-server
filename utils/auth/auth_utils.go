package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"net/http"
	"voice-chat-server/logger"
)

const (
	SecretKey = "ChatServerSecretLey_Macarron"
)

func GetTokenFromRequest(r *http.Request) *jwt.Token {
	token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})

	if err == nil {
		if token.Valid {
			return token
		}
	}
	logger.Logger.Error(err)
	return nil
}

func GetTokenFromRequestParam(r *http.Request) *jwt.Token {
	token, err := request.ParseFromRequest(r, request.ArgumentExtractor{"Authorization"},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})

	if err == nil {
		if token.Valid {
			return token
		}
	}
	logger.Logger.Error(err)
	return nil
}
