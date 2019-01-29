package main

import (
	"fmt"
	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
	"reflect"
	"log"
	"math/rand"
	"time"
)

type (

	// RoomManager represents a component that contains a bundle of room
	RoomManager struct {
		component.Base
		Timer        *nano.Timer
		Rooms        map[int64]*Room
		MatchChannel chan int64      // 匹配队列
		Members      map[int64]*Role // 所有在线玩家

		roomIDSeed int64
	}

	// 提示消息
	NoticeMessage struct {
		Type    int `json:"type"`
		Content string `json:"content"`
	}

	HallMessage struct {
		Type    int `json:"type"`
		Content string `json:"content"`
		Data interface{} `json:"data"`
	}

	GameMessage struct {
		Type    int `json:"type"`
		Content string `json:"content"`
	}
	// NewUser message will be received when new user join room
	NewUser struct {
		Content string `json:"content"`
	}

	stats struct {
		component.Base
		timer         *nano.Timer
		outboundBytes int
		inboundBytes  int
	}
)

func (stats *stats) outbound(s *session.Session, msg nano.Message) error {
	stats.outboundBytes += len(msg.Data)
	return nil
}

func (stats *stats) inbound(s *session.Session, msg nano.Message) error {
	stats.inboundBytes += len(msg.Data)
	return nil
}

func (stats *stats) AfterInit() {
	stats.timer = nano.NewTimer(time.Minute, func() {
		println("OutboundBytes", stats.outboundBytes)
		println("InboundBytes", stats.outboundBytes)
	})
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms:        map[int64]*Room{},
		MatchChannel: make(chan int64, 1000),
		roomIDSeed:   1212,
		Members:      map[int64]*Role{},
	}
}

// AfterInit component lifetime callback
func (mgr *RoomManager) AfterInit() {
	// 退出
	session.Lifetime.OnClosed(func(s *session.Session) {
		if !s.HasKey(roomIDKey) {
			return
		}
		room := s.Value(roomIDKey).(*Room)
		err := room.Group.Leave(s)
		if err != nil {
			log.Println("session leave fail:", err)
		}
	})
	/*mgr.Timer = nano.NewTimer(time.Minute, func() {
		for roomId, room := range mgr.Rooms {
			println(fmt.Sprintf("UserCount: RoomID=%d, Time=%s, Count=%d",
				roomId, time.Now().String(), room.Group.Count()))
		}
	})*/

}
func (mgr *RoomManager) NewRoomId() int64 {
	rand.Seed(time.Now().Unix())
	rnd := rand.Intn(1000) + 1
	mgr.roomIDSeed = (mgr.roomIDSeed + 1) % 10000
	roomId := int64(rnd*10000) + mgr.roomIDSeed
	if _, ok := mgr.Rooms[roomId]; ok {
		roomId = mgr.NewRoomId()
	}
	return roomId
}

// Login
func (mgr *RoomManager) Login(s *session.Session, msg []byte) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	// @todo 检查断线重连

	fmt.Println("login===:", s.UID())
	id := int64(len(mgr.Members) + 1) // @todo 假的用户Id

	role := &Role{id, "testName", "testIcon", 1, 100, 0, s}
	mgr.Members[id] = role
	err := s.Bind(id)
	if err != nil {
		log.Println("====session bind error:", err)
	}

	if err != nil {
		log.Println("====session push error:", err)
	}

	return s.Response(FormatCastRole(role))
}

// 开始匹配
func (mgr *RoomManager) StartMatch(s *session.Session, msg []byte) error {
	if ok := CheckLogin(s); ok{
		mgr.MatchChannel <- s.UID()
	}
	return nil
}

// 大厅操作
func (mgr *RoomManager) Hall(s *session.Session, msg *HallMessage) error {
	fmt.Println("======Msg:", reflect.TypeOf(msg.Data))
	if ok := CheckLogin(s); !ok{
		return nil
	}
	if !s.HasKey(roomIDKey) {
		return fmt.Errorf("not join room yet")
	}
	room := s.Value(roomIDKey).(*Room)
	return room.Group.Broadcast("onMessage", msg)
}

// 游戏房间操作
func (mgr *RoomManager) GameRoom(s *session.Session, msg *GameMessage) error {
	if ok := CheckLogin(s); !ok{
		return nil
	}

	return nil
}