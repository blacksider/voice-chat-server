package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"log"
	"net/http"
	"time"
	"voice-chat-server/dto"
	"voice-chat-server/models"
	"voice-chat-server/service"
)

const (
	SecretKey = "ChatServerSecretLey_Macarron"
)

type AuthController struct {
	UserService *service.ChatUserService
	Session     *service.SessionService
}

func (controller *AuthController) DoLogin(w http.ResponseWriter, r *http.Request) {
	var userDTO dto.UserCredentials
	err := json.NewDecoder(r.Body).Decode(&userDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "Error in request")
		return
	}

	decodedPwd, err := base64.StdEncoding.DecodeString(userDTO.Password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "Error in request")
		return
	}

	var user *models.ChatUser
	user = controller.UserService.AuthUser(userDTO.Username, string(decodedPwd))
	if user == nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "User validate failed")
		return
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := make(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * time.Duration(1)).Unix()
	claims["iat"] = time.Now().Unix()
	token.Claims = claims

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, "Error while signing the token")
		log.Fatal(err)
	}

	controller.Session.Create(user, tokenString)

	response := dto.JwtToken{Token: tokenString}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println("encode failed:", err)
	}
}

func (controller *AuthController) getToken(w http.ResponseWriter, r *http.Request) *jwt.Token {
	token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})

	if err == nil {
		if token.Valid {
			return token
		}
	}
	log.Println(err)
	return nil
}

func (controller *AuthController) ValidateToken(w http.ResponseWriter, r *http.Request) bool {
	token := controller.getToken(w, r)
	if token != nil {
		if token.Valid {
			return true
		}
	}
	return false
}

func (controller *AuthController) GetAuthInfo(w http.ResponseWriter, r *http.Request) {
	jwtToken := controller.getToken(w, r)
	log.Println("Token: ", jwtToken.Raw)
	session := controller.Session.GetByToken(jwtToken.Raw)
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "Token not found")
		return
	}
	user := controller.UserService.GetUserByUsername(session.UserName)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "User not found")
		return
	}
	authInfo := dto.AuthInfo{
		Username:    user.UserName,
		Authorities: make([]dto.AuthAuthority, 0),
	}
	err := json.NewEncoder(w).Encode(&authInfo)
	if err != nil {
		log.Println("encode failed:", err)
	}
}
