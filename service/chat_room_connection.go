package service

import (
	"container/list"
	"encoding/json"
	"errors"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"io"
	"net/http"
	"strconv"
	"time"
	"voice-chat-server/dto"
	"voice-chat-server/logger"
	"voice-chat-server/models"
)

type ChatRoomConn struct {
	Id        string
	Conn      *websocket.Conn
	Context   *ChatRoomConnectionContext
	stop      chan struct{}
	AfterRead func(conn *ChatRoomConn, messageType int, r io.Reader)
}

func (c *ChatRoomConn) listen() {
	c.Conn.SetCloseHandler(func(code int, text string) error {
		msg := websocket.FormatCloseMessage(code, "")
		_ = c.Conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
		for e := c.Context.Connections.Front(); e != nil; e = e.Next() {
			if e.Value.(*ChatRoomConn) == c {
				c.Context.Connections.Remove(e)
				c.Context.ConnectionManager.CleanConnection(c)
				break
			}
		};
		return nil
	})

ReadLoop:
	for {
		select {
		case <-c.stop:
			break ReadLoop
		default:
			msgType, r, err := c.Conn.NextReader()
			if err != nil {
				break ReadLoop
			}
			if c.AfterRead != nil {
				c.AfterRead(c, msgType, r)
			}
		}
	}
}

func (c *ChatRoomConn) Close() error {
	select {
	case <-c.stop:
		return errors.New("conn already been closed")
	default:
		_ = c.Conn.Close()
		close(c.stop)
		logger.Logger.Debug("Close connection", c.Id)
		return nil
	}
}

type ChatRoomConnectionContext struct {
	RoomId            int64
	Connections       *list.List
	ConnectionManager *ChatRoomConnectionManager
}

type ChatRoomConnectionManager struct {
	Upgrader          *websocket.Upgrader
	Session           *SessionService
	ChatServerService *ChatServerService
	contexts          *list.List
	DbService         *DBService
}

func (manager *ChatRoomConnectionManager) Init() error {
	logger.Logger.Info("Init ChatRoomConnectionManager")
	for _, model := range []interface{}{(*models.ChatUserConnStats)(nil)} {
		err := manager.DbService.DB.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists:   true,
			FKConstraints: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func writeErrResponse(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(err.Error()))
}

func (manager *ChatRoomConnectionManager) getContext(room *models.ChatRoom) *ChatRoomConnectionContext {
	if manager.contexts == nil {
		manager.contexts = list.New()
	}

	var handleContext *ChatRoomConnectionContext
	for e := manager.contexts.Front(); e != nil; e = e.Next() {
		if room.Id == e.Value.(*ChatRoomConnectionContext).RoomId {
			handleContext = e.Value.(*ChatRoomConnectionContext)
		}
	}

	if handleContext == nil {
		handleContext = &ChatRoomConnectionContext{
			RoomId:            room.Id,
			Connections:       list.New(),
			ConnectionManager: manager,
		}
		manager.contexts.PushFront(handleContext)
	}
	return handleContext
}

func (manager *ChatRoomConnectionManager) Connect(w http.ResponseWriter, r *http.Request) {
	user := manager.Session.GetUserFromRequestParam(w, r)
	if user == nil {
		return
	}

	roomId := r.FormValue("room")
	roomIdInt, err := strconv.ParseInt(roomId, 10, 64)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
		return
	}

	room, err := manager.ChatServerService.GetRoom(roomIdInt)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
		return
	}

	c, err := manager.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Logger.Error(err)
		writeErrResponse(w, err)
		return
	}

	defer func() {
		_ = c.Close()
	}()

	context := manager.getContext(room)

	var connStat *models.ChatUserConnStats
	err = manager.DbService.DB.RunInTransaction(func(tx *pg.Tx) error {
		connStat, err = manager.AddConnectionData(user, room)
		if err != nil {
			logger.Logger.Error(err)
			return err
		}
		return nil
	})
	if err != nil {
		writeErrResponse(w, errors.New("can not stat connection"))
		return
	}

	newConn := ChatRoomConn{
		Id:        connStat.Id,
		Conn:      c,
		Context:   context,
		stop:      make(chan struct{}),
		AfterRead: manager.handleMessage,
	}

	context.Connections.PushFront(&newConn)
	newConn.listen()

	logger.Logger.Debugf("Connection %s established", newConn.Id)
}

func (manager *ChatRoomConnectionManager) handleMessage(conn *ChatRoomConn, messageType int, r io.Reader) {
	var msg dto.Message
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&msg); err != nil {
		return
	}

	logger.Logger.Debug("Receive message:", msg)

	for e := conn.Context.Connections.Front(); e != nil; e = e.Next() {
		c := e.Value.(*ChatRoomConn)
		if c == conn {
			continue
		}
		err := c.Conn.WriteJSON(msg)
		if err != nil {
			logger.Logger.Error("Failed to send message to connection")
		}
	}
}

func (manager *ChatRoomConnectionManager) AddConnectionData(user *models.ChatUser, room *models.ChatRoom) (*models.ChatUserConnStats, error) {
	var existConn []models.ChatUserConnStats
	err := manager.DbService.DB.Model(&existConn).
		Where("user_id = ?", user.Id).
		Where("room_id = ?", room.Id).
		Select()
	if err != nil {
		logger.Logger.Error(err)
		return nil, err
	}

	if len(existConn) > 0 {
		for _, conn := range existConn {
			manager.CloseConnection(&conn)
		}
	}

	connId := uuid.NewV4().String()
	connStat := models.ChatUserConnStats{
		Id:     connId,
		UserId: user.Id,
		RoomId: room.Id,
	}
	err = manager.DbService.DB.Insert(&connStat)
	if err != nil {
		logger.Logger.Error(err)
		return nil, err
	}
	return &connStat, nil
}

func (manager *ChatRoomConnectionManager) CleanConnection(conn *ChatRoomConn) {
	connId := conn.Id
	_, err := manager.DbService.DB.Model((*models.ChatUserConnStats)(nil)).
		Where("id = ?", connId).
		Delete()
	if err != nil {
		logger.Logger.Error(err)
	}
}

func (manager *ChatRoomConnectionManager) CloseConnection(conn *models.ChatUserConnStats) {
	var handleContext *ChatRoomConnectionContext
	if manager.contexts == nil {
		return
	}
	for e := manager.contexts.Front(); e != nil; e = e.Next() {
		if conn.RoomId == e.Value.(*ChatRoomConnectionContext).RoomId {
			handleContext = e.Value.(*ChatRoomConnectionContext)
			break
		}
	}
	if handleContext == nil || handleContext.Connections == nil {
		return
	}
	for e := handleContext.Connections.Front(); e != nil; e = e.Next() {
		c := e.Value.(*ChatRoomConn)
		if c.Id == conn.Id {
			err := c.Close()
			if err != nil {
				logger.Logger.Error(err)
			}
			handleContext.Connections.Remove(e)
			handleContext.ConnectionManager.CleanConnection(c)
			break
		}
	};
}
