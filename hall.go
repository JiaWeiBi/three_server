package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
)

type (

	// RoomManager represents a component that contains a bundle of room
	RoomManager struct {
		component.Base
		Timer              *nano.Timer
		Rooms              map[int64]*Room
		MatchChannel       chan int64      // 匹配队列
		MatchCancelChannel chan int64      // 匹配取消队列
		Members            map[int64]*Role // 所有在线玩家

		roomIDSeed int64
	}

	// 登录消息
	LoginMessage struct {
		Code     *string   `json:"code"`
		UserInfo *UserInfo `json:"userInfo"`
	}

	// 用户信息
	UserInfo struct {
		NickName  string `json:"nickName"`
		AvatarUrl string `json:"avatarUrl"`
		Gender    uint8  `json:"gender"`
		Province  string `json:"province"`
		City      string `json:"city"`
		Country   string `json:"country"`
	}

	// 提示消息
	NoticeMessage struct {
		Type    int    `json:"type"`
		Content string `json:"content"`
	}

	// 匹配请求消息
	HallMatchMessage struct {
		Type int `json:"type"`
	}

	// 匹配取消返回消息 code 1成功 0失败
	CancelMatchRes struct {
		Code int `json:"code"`
	}

	// 进入好友房间
	HallEnterFriendRoom struct {
		// 好友id
		Fid int64 `json:"fid"`
		// 房间id
		RoomId int64 `json:"roomid"`
	}

	// NewUser message will be received when new user join room
	NewUser struct {
		Info *CastRole `json:"info"`
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
	/*stats.timer = nano.NewTimer(time.Minute, func() {
		println("OutboundBytes", stats.outboundBytes)
		println("InboundBytes", stats.outboundBytes)
	})*/
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms:              map[int64]*Room{},
		MatchChannel:       make(chan int64, 1000),
		MatchCancelChannel: make(chan int64, 1000),
		roomIDSeed:         1212,
		Members:            map[int64]*Role{},
	}
}

// AfterInit component lifetime callback
func (mgr *RoomManager) AfterInit() {
	// 退出
	session.Lifetime.OnClosed(func(s *session.Session) {
		// 设置一分钟后清除内存中的玩家数据, 一分钟内重连需要删除此定时器
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
			}
		}()
		role, ok := GetRoleById(s.UID())
		if ok {
			// 已登录 设置三分钟后清除角色数据1*time.Minute
			log.Println("5miao钟后清除玩家数据==", role.id)
			role.cleanTimer = nano.NewAfterTimer(5*time.Second, func() {
				role.exit()
				// 删除玩家信息
				delete(RoomMgr.Members, role.id)
			})
			if room, ok := RoomMgr.Rooms[role.roomId]; ok {
				err := room.Group.Leave(s)
				if err != nil {
					log.Println("session leave fail:", err)
				}
				return
			}
			role.session = nil
		}
	})
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
func (mgr *RoomManager) Login(s *session.Session, msg *LoginMessage) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	//  验证code
	// #todo test
	//codeRes, err := CheckCode(msg.Code)
	var err error
	if err != nil && false {
		s.Push("notice", "登录失败")
		return nil
	}

	// 是否是新用户
	// #todo test
	//isOlder, role := GetUserInfoByOpenid(codeRes.Openid)
	isOlder, role := GetUserInfoByOpenid(*msg.Code)
	if isOlder {

		//  检查断线重连
		oldRole, isReConn := GetRoleById(role.id)
		log.Println("====", isReConn)
		if !isReConn {
			role.session = s
			mgr.Members[role.id] = role
		} else {
			// 是重连
			oldRole.session = s
			oldRole.cleanTimer.Stop()
			oldRole.cleanTimer = nil
		}
		err = s.Bind(role.id)
		if err != nil {
			log.Println("====session bind error:", err)
			return nil
		}
		err = s.Response(FormatCastRole(role))
		if err != nil {
			log.Println("====session response error:", err)
			return nil
		}

		// 是否处于房间中
		room, hasRoom := mgr.Rooms[mgr.Members[role.id].roomId]
		if hasRoom {
			// 推送房间信息
			room.Join(s, false)
		} else {
			mgr.Members[role.id].roomId = 0
		}
		return nil
	} else {
		// 新用户
		// #todo test
		//role, err := AddNewUser(codeRes.Openid, codeRes.Unionid, msg.UserInfo)
		role, err := AddNewUser(*msg.Code, msg.UserInfo)

		mgr.Members[role.id] = role
		role.session = s
		err = s.Bind(role.id)
		if err != nil {
			log.Println("====session bind error:", err)
			panic(err)
		}
		return s.Response(FormatCastRole(role))
	}
}

// 开始匹配
func (mgr *RoomManager) StartMatch(s *session.Session, msg *HallMatchMessage) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	role, _ := GetRoleById(s.UID())
	if role.status != 0 {
		log.Println("====进入匹配失败 status:", role.status)
		return nil
	}
	switch msg.Type {
	// 训练赛
	case 0:
		if ok := CheckLogin(s); ok {
			mgr.MatchChannel <- s.UID()
		}
		s.Response("ok")
		break
	case 1: //好友赛
		roomId := mgr.NewRoomId()
		room := NewRoom(roomId)
		room.FPlayer = s.UID()
		mgr.Rooms[roomId] = room
		room.Init(1)
		break
	case 2:
		break
	}
	role.status = 1
	return nil
}

// 进入好友赛
func (mgr *RoomManager) EnterFriendRoom(s *session.Session, msg *HallEnterFriendRoom) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	if room, ok := mgr.Rooms[msg.RoomId]; ok {
		if room.FPlayer != 0 && room.SPlayer != 0 {
			s.Response("房间已满")
			return nil
		} else if room.FPlayer == msg.Fid {
			room.SPlayer = s.UID()
			Role, _ := GetRoleById(room.SPlayer)

			Role.roomId = room.Id
			Role.status = 1
			room.Join(s, true)
			s.Response("ok")
		} else if room.SPlayer == msg.Fid {
			room.FPlayer = s.UID()
			Role, _ := GetRoleById(room.FPlayer)

			Role.roomId = room.Id
			Role.status = 1
			room.Join(s, true)
			s.Response("ok")
		} else {
			s.Response("房间已过期")
			return nil
		}
	}
	return nil
}

// 取消匹配
func (mgr *RoomManager) CancelMatch(s *session.Session, msg *HallMatchMessage) error {
	if ok := CheckLogin(s); ok {
		switch msg.Type {
		case 0:
			mgr.MatchCancelChannel <- s.UID()
			break
		case 2:
			break
		default:
			break
		}
	}
	return nil
}

// ==================================调试接口，不开放================================
// 获取房间数量
func (mgr *RoomManager) RoomNum(s *session.Session, msg []byte) error {
	return s.Response(len(RoomMgr.Rooms))
}

// 获取在线玩家数量
func (mgr *RoomManager) RoleNum(s *session.Session, msg []byte) error {
	return s.Response(len(RoomMgr.Members))
}

// 获取房间信息
func (mgr *RoomManager) RoomInfo(s *session.Session, msg *struct{ Id int64 }) error {
	if room, ok := RoomMgr.Rooms[msg.Id]; ok {
		return s.Response(room)
	} else {
		return s.Response("fail")
	}
}

// 获取玩家信息
func (mgr *RoomManager) RoleInfo(s *session.Session, msg *struct{ Id int64 }) error {
	if role, ok := RoomMgr.Members[msg.Id]; ok {
		res := make(map[string]interface{})
		res["roomId"] = role.roomId
		res["status"] = role.status
		return s.Response(res)
	} else {
		return s.Response("fail")
	}
}

//获取匹配队列
func (mgr *RoomManager) MatchSlice(s *session.Session, msg []byte) error {
	return s.Response(matchSlice)
}
