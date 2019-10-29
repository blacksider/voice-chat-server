package service

import (
	"container/list"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
	"voice-chat-server/dto"
	"voice-chat-server/models"
)

type ChatRoomConn struct {
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
		return nil
	}
}

type ChatRoomConnectionContext struct {
	RoomId      int64
	Connections *list.List
}

type ChatRoomConnectionManager struct {
	Upgrader          *websocket.Upgrader
	ChatServerService *ChatServerService
	contexts          *list.List
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
			RoomId:      room.Id,
			Connections: list.New(),
		}
		manager.contexts.PushFront(handleContext)
	}
	return handleContext
}

func (manager *ChatRoomConnectionManager) Connect(w http.ResponseWriter, r *http.Request) {
	roomId := r.FormValue("room")
	roomIdInt, err := strconv.ParseInt(roomId, 10, 64)
	if err != nil {
		log.Println(err)
		writeErrResponse(w, err)
		return
	}
	room, err := manager.ChatServerService.GetRoom(roomIdInt)
	if err != nil {
		log.Println(err)
		writeErrResponse(w, err)
		return
	}

	c, err := manager.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		writeErrResponse(w, err)
		return
	}

	defer func() {
		_ = c.Close()
	}()

	context := manager.getContext(room)

	newConn := ChatRoomConn{
		Conn:      c,
		Context:   context,
		stop:      make(chan struct{}),
		AfterRead: manager.handleMessage,
	}
	context.Connections.PushFront(&newConn)
	newConn.listen()
}

func (manager *ChatRoomConnectionManager) handleMessage(conn *ChatRoomConn, messageType int, r io.Reader) {
	var msg dto.Message
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&msg); err != nil {
		return
	}

	for e := conn.Context.Connections.Front(); e != nil; e = e.Next() {
		c := e.Value.(*ChatRoomConn)
		log.Println(c == conn)
		if c == conn {
			continue
		}
		err := c.Conn.WriteJSON(msg)
		if err != nil {
			log.Println("failed to send message connection")
		}
	}
}
