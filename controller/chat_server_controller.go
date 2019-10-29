package controller

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"voice-chat-server/logger"
	"voice-chat-server/service"
)

type ChatServerController struct {
	ChatServerService *service.ChatServerService
}

func (controller *ChatServerController) ListServers(w http.ResponseWriter, r *http.Request) {
	serverList := controller.ChatServerService.ListServers()
	err := json.NewEncoder(w).Encode(serverList)
	if err != nil {
		logger.Logger.Error("encode failed:", err)
	}
}

func writeErrResponse(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(err.Error()))
}

func (controller *ChatServerController) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
		return
	}
	serverInfo, err := controller.ChatServerService.GetServerInfo(idInt)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
		return
	}
	err = json.NewEncoder(w).Encode(serverInfo)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
	}
}

func (controller *ChatServerController) ListRooms(w http.ResponseWriter, r *http.Request) {
	serverId := r.FormValue("id")

	idInt, err := strconv.ParseInt(serverId, 10, 64)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
		return
	}
	roomList, err := controller.ChatServerService.ListRooms(idInt)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
		return
	}
	err = json.NewEncoder(w).Encode(roomList)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
	}
}
