package service

import (
	"errors"
	"github.com/go-pg/pg/v9/orm"
	"log"
	"voice-chat-server/models"
)

type ChatServerService struct {
	DbService *DBService
}

func (service *ChatServerService) Init() error {
	for _, model := range []interface{}{(*models.ChatServer)(nil), (*models.ChatRoom)(nil)} {
		err := service.DbService.DB.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists:   true,
			FKConstraints: true,
		})
		if err != nil {
			return err
		}
	}

	server := service.GetServerById(1)

	var serverInfo models.ChatServer
	if server == nil {
		serverInfo = models.ChatServer{
			Id:          1,
			Name:        "Default server",
			Description: "Default server",
		}
		err := service.DbService.DB.Insert(&serverInfo)
		if err != nil {
			return err
		}

		room := service.GetRoomById(1)
		var roomInfo models.ChatRoom
		if room == nil {
			roomInfo = models.ChatRoom{
				Id:          1,
				Name:        "Default room",
				Description: "Default room",
				ServerId:    serverInfo.Id,
			}
			err = service.DbService.DB.Insert(&roomInfo)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (service *ChatServerService) GetServerById(id int64) *models.ChatServer {
	var server models.ChatServer
	err := service.DbService.DB.Model(&server).Where("id = ?", id).Select()
	if err != nil {
		log.Println(err)
		return nil
	}
	return &server
}

func (service *ChatServerService) GetRoomById(id int64) *models.ChatRoom {
	var room models.ChatRoom
	err := service.DbService.DB.Model(&room).Where("id = ?", id).Select()
	if err != nil {
		log.Println(err)
		return nil
	}
	return &room
}

func (service *ChatServerService) ListServers() []models.ChatServerData {
	var serverList []models.ChatServerData

	var servers []models.ChatServer
	err := service.DbService.DB.Model(&servers).Select()
	if err != nil {
		log.Println(err)
		return serverList
	}
	for _, server := range servers {
		serverData := models.ChatServerData{
			Id:          server.Id,
			Name:        server.Name,
			Description: server.Description,
		}
		serverList = append(serverList, serverData)
	}
	return serverList
}

func (service *ChatServerService) GetServerInfo(serveId int64) (*models.ChatServerData, error) {
	serverInfo := service.GetServerById(serveId)
	if serverInfo == nil {
		return nil, errors.New("can not find server")
	}
	var serverData = models.ChatServerData{
		Id:          serverInfo.Id,
		Name:        serverInfo.Name,
		Description: serverInfo.Description,
	}
	return &serverData, nil
}

func (service *ChatServerService) ListRooms(serverId int64) ([]models.ChatRoom, error) {
	var rooms []models.ChatRoom
	err := service.DbService.DB.Model(&rooms).
		Column("chat_room.*").
		Relation("Server").
		Join("join chat_server as svr").
		JoinOn("svr.id = chat_room.server_id").
		JoinOn("svr.id = ?", serverId).
		Select()
	if err != nil {
		log.Println(err)
	}
	if err != nil || rooms == nil {
		return nil, errors.New("can not find rooms")
	}
	return rooms, nil
}

func (service *ChatServerService) GetRoom(roomId int64) (*models.ChatRoom, error) {
	roomInfo := service.GetRoomById(roomId)
	if roomInfo == nil {
		return nil, errors.New("can not find room")
	}
	return roomInfo, nil
}
