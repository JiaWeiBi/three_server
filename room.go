package main

import (
	"fmt"
	"github.com/lonng/nano"
	"github.com/lonng/nano/session"
	"log"
)

const (
	roomIDKey = "ROOM_ID"
)

type (
	Room struct {
		Id    int64 // Id
		Group *nano.Group
		Timer *nano.Timer
		Woner uint8 // 1:fPlayer 2:sPlayer
		// 房间状态
		//0:双方未准备 1:1号准备 2:2号准备
		// 11:FPlayer下子 12:FPlayer走子 13:FPlayer揪子 21:SPlayer下子 22:SPlayer走子 23:SPlayer揪子
		Status uint8
		// 棋盘表 0:为空 1：FPlayer棋子 2：SPlayer棋子 3：揪过棋子
		PointMap map[Point]int

		FPlayer, SPlayer             int64
		FPlayerHelper, SPlayerHelper int64
		WatchList                    []int64
		//
		//HotPoints []*Point
		//
		StepList []*Step
	}

	// 房间信息 用于推送
	CastRoomInfo struct {
		Woner         uint8         `json:"woner"`
		Status        uint8         `json:"status"`
		PointMap      map[string]int `json:"pointmap"`
		Flag          int           `json:"flag"`
		FPlayer       *CastRole      `json:"fplayer"`
		SPlayer       *CastRole      `json:"splayer"`
		FPlayerHelper *CastRole      `json:"fhelper"`
		SPlayerHelper *CastRole      `json:"shelper"`
	}

	Point struct {
		X int
		Y int
	}
	Step struct {
		// 1:下子 2:走子 3:揪子
		Type int
		Src  *Point
		Dest *Point
	}
)

var Points []Point

func init() {
	Points = []Point{
		{-3, -3}, {-3, 0}, {-3, 3},
		{-2, -2}, {-2, 0}, {-2, 2},
		{-1, -1}, {-1, 0}, {-1, 1},
		{0, -3}, {0, -2}, {0, -1}, {0, 1}, {0, 2}, {0, 3},
		{1, -1}, {1, 0}, {1, 1},
		{2, -2}, {2, 0}, {2, 2},
		{3, -3}, {3, 0}, {3, 3},
	}
}
func init() {
	Map := make(map[Point]int)
	for _, p := range Points {
		Map[p] = 0
	}
}

func NewRoom(roomId int64) *Room {
	room := &Room{}
	room.Id = roomId
	room.Status = 0

	room.Woner = 1
	room.PointMap = make(map[Point]int)
	for _, p := range Points {
		room.PointMap[p] = 0
	}
	room.Group = nano.NewGroup("room")
	return room
}

// 房间初始化
func (room *Room) Init() {
	fmt.Println("房间初始化,Id:", room.Id)

	fRole, _ := GetRoleById(room.FPlayer)
	sRole, _ := GetRoleById(room.SPlayer)
	fRole.roomId = room.Id
	sRole.roomId = room.Id
	log.Println(*fRole.session, *sRole.session)
	room.Join(fRole.session, false)
	room.Join(sRole.session, false)
}

func (p *Point)ToString() string{
	return fmt.Sprintf("{X:%d,Y:%d}", p.X, p.Y)
}

// joinType: 1:1号玩家（房主） 2:2号玩家 3:1号Helper 4:1号Helper 5:旁观
func (room *Room) Join(s *session.Session, cast bool) (bool, string) {
	s.Set(roomIDKey, room)
	// @todo 推送房间信息
	err := s.Push("onRoomInfo", room.getCastRoomInfo())
	if err != nil {
		return false, "推送房间信息失败"
	}
	// notify others
	if cast {
		err = room.Group.Broadcast("onNewUser", &NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
		if err != nil {
			log.Panicln("通知房间内其他玩家失败", err)
		}
	}

	// new user join group
	err = room.Group.Add(s) // add session to group
	if err != nil {
		return false, "加入房间分组失败"
	}
	return true, ""
}

// 获取房间信息
func (room *Room) getCastRoomInfo() *CastRoomInfo {
	info := &CastRoomInfo{}
	info.Woner = room.Woner
	info.Status = room.Status

	if room.FPlayer > 0 {
		fp, _ := GetRoleById(room.FPlayer)
		info.FPlayer = FormatCastRole(fp)
	}
	if room.FPlayerHelper > 0 {
		fph, _ := GetRoleById(room.FPlayerHelper)
		info.FPlayerHelper = FormatCastRole(fph)
	}
	if room.SPlayer > 0 {
		sp, _ := GetRoleById(room.SPlayer)
		info.SPlayer = FormatCastRole(sp)
	}
	if room.SPlayerHelper > 0 {
		sph, _ := GetRoleById(room.SPlayerHelper)
		info.SPlayerHelper = FormatCastRole(sph)
	}
	info.PointMap = map[string]int{}
	for p, v := range room.PointMap{
		info.PointMap[p.ToString()] = v
	}
	return info
}

//++++++++++++++++++++++++++++++++++++++++++
// 游戏逻辑
//++++++++++++++++++++++++++++++++++++++++++

func (room *Room)Ready(role *Role, msg []byte) error{

	return nil
}