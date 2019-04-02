package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
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
		// 房间类型 0:普通匹配 1:好友赛 2;段位赛
		Type int
		// 房间状态
		//0:双方未准备 1:1号准备 2:2号准备
		// 11:FPlayer下子 12:FPlayer走子 13:FPlayer揪子 21:SPlayer下子 22:SPlayer走子 23:SPlayer揪子
		Status uint8
		// 倍率
		Magnification int
		// 求和人 0：没有人求和 1:一号求和 2:二号求和
		SuePeace int
		// 棋盘表 0:为空 1：FPlayer棋子 2：SPlayer棋子 3：揪过棋子
		PointMap map[Point]int

		FPlayer, SPlayer             int64
		FPlayerHelper, SPlayerHelper int64
		WatchList                    []int64
		//
		//HotPoints []*Point
		//
		StepList []*Step
		ChessNum int // 总共的下子数量，揪子不减少
		// 超时次数 连续第三次超时判定为输
		// 状态开始的时间戳
		ActionTime int64
	}

	// 房间信息 用于推送
	CastRoomInfo struct {
		Id int64 `json:"id"`
		// 房间状态
		//0:双方未准备 1:1号准备 2:2号准备
		// 11:FPlayer下子 12:FPlayer走子 13:FPlayer揪子 21:SPlayer下子 22:SPlayer走子 23:SPlayer揪子
		Status   uint8          `json:"status"`
		PointMap map[string]int `json:"pointmap"`
		// 房间类型 0:普通匹配 1:好友赛 2;段位赛
		Type int `json:"type"`
		// 倍率
		Magnification int `json:"mag"`
		// 求和人 0：没有人求和 1:一号求和 2:二号求和
		SuePeace   int   `json:"peace"`
		ActionTime int64 `json:"actiontime"`

		FPlayer       *CastRole `json:"fplayer"`
		SPlayer       *CastRole `json:"splayer"`
		FPlayerHelper *CastRole `json:"fhelper"`
		SPlayerHelper *CastRole `json:"shelper"`
	}

	// 房间状态广播
	StatusChange struct {
		Status    uint8 `json:"status"`
		StartTime int64 `json:"starttime"`
	}

	Point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	Step struct {
		// 1:下子 2:走子 3:揪子
		Type   int    `json:"type"`
		Player int64  `json:"player"`
		Src    *Point `json:"src"`
		Dst    *Point `json:"dst"`
	}

	// 结算信息
	SettleMsg struct {
		Winner    int `json:"winner"`
		WinGold   int `json:"wingold"`
		WinScore  int `json:"winscore"`
		LossGold  int `json:"lossgold"`
		LossScore int `json:"lossscore"`
	}
)

var Points []Point

// 成三校验的点位列表
var C3CheckMap map[Point][][]Point

// 每个点相邻的点位列表
var BorderPointMap map[Point][]Point

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
	C3CheckMap = make(map[Point][][]Point)
	testFun := func(slice []Point, p Point) []Point {
		if len(slice) == 3 {
			return slice
		} else if len(slice) == 6 {
			newSlice := []Point{}
			for _, p2 := range slice {
				if (p.X == 0 && p.Y*p2.Y > 0) || (p.Y == 0 && p.X*p2.X > 0) {
					newSlice = append(newSlice, p2)
				}
			}
			return newSlice
		} else {
			panic("初始化成三出错")
		}
	}
	for _, p := range Points {
		XSlice := []Point{}
		for _, p2 := range Points {
			if p.X == p2.X {
				XSlice = append(XSlice, p2)
			}
		}
		YSlice := []Point{}
		for _, p2 := range Points {
			if p.Y == p2.Y {
				YSlice = append(YSlice, p2)
			}
		}

		C3CheckMap[p] = [][]Point{testFun(XSlice, p), testFun(YSlice, p)}
	}

	borderFun := func(p1 *Point, p2 *Point) bool {
		//校验是否相邻
		if p1.X == 0 {
			//原点是Y轴点
			if p2.X == 0 && (p1.Y-p2.Y == 1 || p1.Y-p2.Y == -1) {
				return true
			} else if p2.X != 0 && p1.Y == p2.Y {
				return true
			}
		} else if p1.Y == 0 {
			//原点是X轴点
			if p2.Y == 0 && (p1.X-p2.X == 1 || p1.X-p2.X == -1) {
				return true
			} else if p2.Y != 0 && p1.X == p2.X {
				return true
			}
		} else {
			// 原点是角点
			if p2.X == 0 && p1.Y == p2.Y {
				return true
			} else if p2.Y == 0 && p1.X == p2.X {
				return true
			}
		}
		return false
	}
	BorderPointMap = make(map[Point][]Point)
	for _, p1 := range Points {
		for _, p2 := range Points {
			if borderFun(&p1, &p2) {
				BorderPointMap[p1] = append(BorderPointMap[p1], p2)
			}
		}
	}
}

func NewRoom(roomId int64) *Room {
	room := &Room{}
	room.Id = roomId
	room.Status = 0

	room.Woner = 1
	room.Magnification = 1
	room.PointMap = make(map[Point]int)
	for _, p := range Points {
		room.PointMap[p] = 0

	}
	room.Group = nano.NewGroup(fmt.Sprintf("%d", roomId))
	return room
}

// 房间初始化
func (room *Room) Init(t int) {
	room.Type = t
	switch t {
	case 0: //普通匹配
		fRole, _ := GetRoleById(room.FPlayer)
		sRole, _ := GetRoleById(room.SPlayer)
		fRole.roomId = room.Id
		sRole.roomId = room.Id

		room.Join(fRole.session, false)
		room.Join(sRole.session, false)
		break
	case 1: //好友战
		fRole, _ := GetRoleById(room.FPlayer)

		fRole.roomId = room.Id

		room.Join(fRole.session, false)
		break
	case 2: // 段位赛
		break
	}
}

func (p *Point) ToString() string {
	return fmt.Sprintf("{X:%d,Y:%d}", p.X, p.Y)
}

// cast: 是否通知房间内的其他玩家
func (room *Room) Join(s *session.Session, cast bool) (bool, string) {
	// notify others
	if cast {
		role, _ := GetRoleById(s.UID())
		err := room.Group.Broadcast("onNewUser", FormatCastRole(role))
		if err != nil {
			log.Panicln("通知房间内其他玩家失败", err)
		}
	}
	if s != nil {
		err := s.Push("onRoomInfo", room.getCastRoomInfo())
		if err != nil {
			return false, "推送房间信息失败"
		}
		err = room.Group.Add(s) // add session to group
		if err != nil {
			return false, "加入房间分组失败"
		}
	}

	return true, ""
}

// 获取房间信息
func (room *Room) getCastRoomInfo() *CastRoomInfo {
	info := &CastRoomInfo{}
	info.Status = room.Status
	info.Id = room.Id
	info.Type = room.Type
	info.Magnification = room.Magnification
	info.SuePeace = room.SuePeace
	info.ActionTime = room.ActionTime

	if fp, ok := GetRoleById(room.FPlayer); ok {
		info.FPlayer = FormatCastRole(fp)
	}
	if fph, ok := GetRoleById(room.FPlayerHelper); ok {

		info.FPlayerHelper = FormatCastRole(fph)
	}
	if sp, ok := GetRoleById(room.SPlayer); ok {
		info.SPlayer = FormatCastRole(sp)
	}
	if sph, ok := GetRoleById(room.SPlayerHelper); ok {
		info.SPlayerHelper = FormatCastRole(sph)
	}
	info.PointMap = map[string]int{}
	for p, v := range room.PointMap {
		info.PointMap[p.ToString()] = v
	}
	return info
}

// 开始
func (room *Room) Start() {
	// 数据初始化
	for p := range room.PointMap {
		room.PointMap[p] = 0
	}
	// 一号玩家开始下子
	room.ChangeStatus(11)
}

// 成三校验
func (room *Room) CheckThree(p *Point) bool {
	checkList := C3CheckMap[*p]
	if !(room.PointMap[*p] == 1 || room.PointMap[*p] == 2) {
		return false
	}
	for _, list := range checkList {
		if room.PointMap[list[0]] == room.PointMap[list[1]] && room.PointMap[list[0]] == room.PointMap[list[2]] {
			return true
		}
	}
	return false
}

// 揪子校验
func (room *Room) CheckTake(p *Point) bool {
	switch room.Status {
	case 13:
		if room.PointMap[*p] != 2 {
			return false
		}
		break
	case 23:
		if room.PointMap[*p] != 1 {
			return false
		}
		break
	}
	// 成三的棋子只有在场上没有不成三的棋子时才能揪
	if room.CheckThree(p) {
		for point, v := range room.PointMap {
			if v == room.PointMap[*p] && !room.CheckThree(&point) {
				return false
			}
		}
	}
	return true
}

// 走子校验
func (room *Room) CheckMove(step *Step) bool {
	if step.Type != 2 || (room.Status != 12 && room.Status != 22) {
		return false
	}
	if v, ok := room.PointMap[*step.Src]; !ok || v != int(room.Status/10) {
		return false
	}
	if v, ok := room.PointMap[*step.Dst]; !ok || (v == 1 || v == 2) {
		return false
	}
	//校验是否相邻
	if step.Src.X == 0 {
		//原点是Y轴点
		if step.Dst.X == 0 && (step.Src.Y-step.Dst.Y == 1 || step.Src.Y-step.Dst.Y == -1) {
			return true
		} else if step.Dst.X != 0 && step.Src.Y == step.Dst.Y {
			return true
		}
	} else if step.Src.Y == 0 {
		//原点是X轴点
		if step.Dst.Y == 0 && (step.Src.X-step.Dst.X == 1 || step.Src.X-step.Dst.X == -1) {
			return true
		} else if step.Dst.Y != 0 && step.Src.X == step.Dst.X {
			return true
		}
	} else {
		// 原点是角点
		if step.Dst.X == 0 && step.Src.Y == step.Dst.Y {
			return true
		} else if step.Dst.Y == 0 && step.Src.X == step.Dst.X {
			return true
		}
	}
	return false
}

// 输赢判定 走子状态时才判定
func (room *Room) CheckWin() bool {
	if room.Status < 10 || room.Status%10 != 2 {
		return false
	}
	Winner := 0
	var l1, l2 []Point
	for _, p := range Points {
		if room.PointMap[p] == 1 {
			l1 = append(l1, p)
		} else if room.PointMap[p] == 2 {
			l2 = append(l2, p)
		}
	}

	switch room.Status {
	case 12:
		if len(l1) < 3 {
			Winner = 2
		} else {
			flag := true
			for _, p := range l1 {
				for _, bp := range BorderPointMap[p] {
					if room.PointMap[bp] == 0 || room.PointMap[bp] == 3 {
						flag = false
						break
					}
				}
				if !flag {
					break
				}
			}
			if flag {
				Winner = 2
			}
		}

		break
	case 22:
		if len(l2) < 3 {
			Winner = 1
		} else {
			flag := true
			for _, p := range l2 {
				for _, bp := range BorderPointMap[p] {
					if room.PointMap[bp] == 0 || room.PointMap[bp] == 3 {
						flag = false
						break
					}
				}
				if !flag {
					break
				}
			}
			if flag {
				Winner = 1
			}
		}
		break
	}
	// @todo 输赢处理
	if Winner == 0 {
		return false
	}
	room.Settle(Winner)

	return true
}

// @todo 结算积分
func (room *Room) Settle(winner int) *SettleMsg {
	res := &SettleMsg{Winner: winner}
	res.LossGold = 20 * room.Magnification
	res.WinGold = 15 * room.Magnification
	if room.Type == 2 {

	}
	switch winner {
	case 1:
		role1, _ := GetRoleById(room.FPlayer)
		//role1.AddGold(res.WinGold)
		role1.AddGold(0)
		role2, _ := GetRoleById(room.SPlayer)
		//role2.AddGold(res.LossGold)
		role2.AddGold(0)
		break
	case 2:
		role1, _ := GetRoleById(room.SPlayer)
		//role1.AddGold(res.WinGold)
		role1.AddGold(0)
		role2, _ := GetRoleById(room.FPlayer)
		//role2.AddGold(res.LossGold)
		role2.AddGold(0)
		break
	}
	// 停止超时计时器
	room.Timer.Stop()
	room.Cast("onSettle", res)
	room.ChangeStatus(0)
	room.ChessNum = 0

	for p := range room.PointMap {
		room.PointMap[p] = 0
	}
	return res
}

//@todo  超时处理
func (room *Room) TimeOut() {
	if room.Status > 10 {
		// 如果步骤超时当认输处理
		var winner int
		if room.Status/10 == 1 {
			winner = 2
		} else {
			winner = 1
		}
		room.Settle(winner)

	} else {
		room.Timer.Stop()
		room.Timer = nil
	}
}

// 房间状态修改并广播
func (room *Room) ChangeStatus(s uint8) {
	room.Status = s
	now := time.Now().Unix()
	room.ActionTime = now
	room.Cast("onStatus", &StatusChange{room.Status, room.ActionTime})
	if room.Status%10 == 2 {
		room.CheckWin()
	}
	if s > 10 {
		if room.Timer == nil {
			// 设置超时计时器
			room.Timer = nano.NewAfterTimer(1*time.Minute, room.TimeOut)
		} else {
			room.Timer.Stop()
			// 设置超时计时器
			room.Timer = nano.NewAfterTimer(1*time.Minute, room.TimeOut)
		}
	}
}

// 广播步骤
func (room *Room) CastStep(step *Step) {
	err := room.Group.Broadcast("onStep", step)
	if err != nil {
		log.Println(err)
	}
}

// 广播
func (room *Room) Cast(router string, data interface{}) {
	err := room.Group.Broadcast(router, data)
	if err != nil {
		log.Println(err)
	}
}

//++++++++++++++++++++++++++++++++++++++++++
// 游戏逻辑
//++++++++++++++++++++++++++++++++++++++++++
type RoomHandlers struct {
	component.Base
}

// 准备
func (self *RoomHandlers) Ready(s *session.Session, msg []byte) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, msg)
	uid := s.UID()
	role, _ := GetRoleById(uid)
	roomId := role.roomId
	room, hasRoom := RoomMgr.Rooms[roomId]
	if !hasRoom {
		panic("当前未加入游戏")
	}

	switch room.Status {
	case 0:
		if room.FPlayer == uid {
			room.ChangeStatus(1)
		} else if room.SPlayer == uid {
			room.ChangeStatus(2)
		}
		break
	case 1:
		if room.SPlayer == uid {
			room.Start()
		}
		break
	case 2:
		if room.FPlayer == uid {
			room.Start()
		}
		break
	default:
		Response(s, "fail")
		return nil
	}
	Response(s, "ok")
	return nil
}
func (self *RoomHandlers) CancelReady(s *session.Session, msg []byte) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, msg)
	uid := s.UID()
	role, _ := GetRoleById(uid)
	roomId := role.roomId
	room, hasRoom := RoomMgr.Rooms[roomId]
	if !hasRoom {
		panic("当前未加入游戏")
	}
	switch room.Status {
	case 1:
		if room.FPlayer == uid {
			room.ChangeStatus(0)
			Response(s, "ok")
		}
		return nil
	case 2:
		if room.SPlayer == uid {
			room.ChangeStatus(0)
			Response(s, "ok")
		}
		return nil
	}
	Response(s, "fail")
	return nil
}

// 落子
func (self *RoomHandlers) Put(s *session.Session, p *Point) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, "落子")
	uid := s.UID()
	role, _ := GetRoleById(uid)
	roomId := role.roomId
	room, hasRoom := RoomMgr.Rooms[roomId]
	if !hasRoom {
		panic("当前未加入游戏")
	}
	switch room.Status {
	case 11:
		// 一号玩家落子
		if room.FPlayer != uid {
			Response(s, "fail")
			return nil
		}
		if room.PointMap[*p] != 0 {
			role.Push("notice", NoticeMessage{Content: "当前位置不能落子"})
			return nil
		}
		room.ChessNum++
		room.PointMap[*p] = 1

		if room.CheckThree(p) {
			// 成三了
			room.ChangeStatus(13)
		} else {
			if room.ChessNum == 18 {
				room.ChangeStatus(22)
			} else {
				room.ChangeStatus(21)
			}
		}
		Response(s, "ok")
		break
	case 21:
		// 二号玩家落子
		if room.SPlayer != uid {
			Response(s, "fail")
			return nil
		}
		if room.PointMap[*p] != 0 {
			role.Push("notice", NoticeMessage{Content: "当前位置不能落子"})
			return nil
		}
		room.ChessNum++
		room.PointMap[*p] = 2

		if room.CheckThree(p) {
			// 成三了
			room.ChangeStatus(23)
		} else {
			if room.ChessNum == 18 {
				room.ChangeStatus(12)
			} else {
				room.ChangeStatus(11)
			}
		}
		Response(s, "ok")
		break
	default:
		Response(s, "fail")
		return nil
	}
	// 广播
	step := &Step{
		1,
		uid,
		p,
		nil,
	}
	room.StepList = append(room.StepList, step)
	room.CastStep(step)
	return nil
}

// 揪子
func (self *RoomHandlers) Take(s *session.Session, p *Point) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, "揪子")
	uid := s.UID()
	role, _ := GetRoleById(uid)
	roomId := role.roomId
	room, hasRoom := RoomMgr.Rooms[roomId]
	if !hasRoom {
		panic("当前未加入游戏")
	}
	switch room.Status {
	case 13:
		if room.FPlayer != uid {
			Response(s, "fail")
			return nil
		}
		if !room.CheckTake(p) {
			role.Push("notice", NoticeMessage{Content: "此位置不能揪子"})
			return nil
		}
		room.PointMap[*p] = 3
		// 广播
		step := &Step{
			3,
			uid,
			p,
			nil,
		}
		room.StepList = append(room.StepList, step)
		room.CastStep(step)
		if room.ChessNum == 18 {
			room.ChangeStatus(22)
		} else {
			room.ChangeStatus(21)
		}
		Response(s, "ok")
		break
	case 23:
		if room.SPlayer != uid {
			Response(s, "fail")
			return nil
		}
		if !room.CheckTake(p) {
			role.Push("notice", NoticeMessage{Content: "此位置不能揪子"})
			return nil
		}
		room.PointMap[*p] = 3
		// 广播
		step := &Step{
			3,
			uid,
			p,
			nil,
		}
		room.StepList = append(room.StepList, step)
		room.CastStep(step)
		if room.ChessNum == 18 {
			room.ChangeStatus(12)
		} else {
			room.ChangeStatus(11)
		}
		Response(s, "ok")
		break
	}
	return nil
}

// 走子
func (self *RoomHandlers) Move(s *session.Session, step *Step) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, "走子")
	uid := s.UID()
	role, _ := GetRoleById(uid)
	roomId := role.roomId
	room, hasRoom := RoomMgr.Rooms[roomId]
	if !hasRoom {
		panic("当前未加入游戏")
	}
	switch room.Status {
	case 12:
		if !room.CheckMove(step) {
			return nil
		}
		room.PointMap[*step.Dst] = 1
		room.PointMap[*step.Src] = 0
		if room.CheckThree(step.Dst) {
			// 成三了
			room.ChangeStatus(13)
		} else {
			room.ChangeStatus(22)
		}
		Response(s, "ok")
		break
	case 22:
		if !room.CheckMove(step) {
			return nil
		}
		room.PointMap[*step.Dst] = 2
		room.PointMap[*step.Src] = 0

		if room.CheckThree(step.Dst) {
			// 成三了
			room.ChangeStatus(23)
		} else {
			room.ChangeStatus(12)
		}
		Response(s, "ok")
		break
	default:
		role.Push("notice", `错误状态:${room.Status}`)
		return nil
	}
	// 广播
	room.StepList = append(room.StepList, step)
	room.CastStep(step)
	return nil
}

// 认输
func (self *RoomHandlers) Fail(s *session.Session, msg []byte) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, "认输")
	role, _ := GetRoleById(s.UID())
	if room, ok := RoomMgr.Rooms[role.roomId]; ok && room.Status > 10 {
		var winner int
		if role.id == room.FPlayer {
			winner = 2
		} else {
			winner = 1
		}
		room.Settle(winner)
	}
	return nil
}

// 求和
func (self *RoomHandlers) SuePeace(s *session.Session, msg struct{ Code int }) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, "求和")
	role, _ := GetRoleById(s.UID())
	if room, ok := RoomMgr.Rooms[role.roomId]; ok && room.Status > 10 && room.SuePeace == 0 && msg.Code == 1 {
		if room.SPlayer == role.id {
			room.SuePeace = 2
			FP, _ := GetRoleById(room.FPlayer)
			FP.Push("suePeace", msg)
		} else if room.FPlayer == role.id {
			room.SuePeace = 1
			SP, _ := GetRoleById(room.SPlayer)
			SP.Push("suePeace", msg)
		}
	} else if room, ok := RoomMgr.Rooms[role.roomId]; ok && room.Status > 10 && room.SuePeace != 0 && msg.Code == 0 {
		// 取消求和
		if room.SPlayer == role.id && room.SuePeace == 2 {
			room.SuePeace = 0
			FP, _ := GetRoleById(room.FPlayer)
			FP.Push("cancelSuePeace", msg)
		} else if room.FPlayer == role.id && room.SuePeace == 1 {
			room.SuePeace = 0
			SP, _ := GetRoleById(room.SPlayer)
			SP.Push("cancelSuePeace", msg)
		}
	} else {
		//
		// \Response(s,"fail")
	}
	return nil
}

// 回应求和
func (self *RoomHandlers) ResSuePeace(s *session.Session, msg struct{ Code int }) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, "回应求和")
	role, _ := GetRoleById(s.UID())
	if room, ok := RoomMgr.Rooms[role.roomId]; ok && room.Status > 10 {
		if (room.SPlayer == role.id && room.SuePeace == 1) || (room.FPlayer == role.id && room.SuePeace == 2) {
			switch msg.Code {
			case 0: // 拒绝
				room.SuePeace = 0
				break
			default: // 同意
				break
			}
		} else {
			//Response(s,"fail")
		}
	}
	return nil
}

// 退出房间
func (self *RoomHandlers) Quit(s *session.Session, msg []byte) error {
	if ok := CheckLogin(s); !ok {
		return nil
	}
	defer self.recover(s, "退出房间")
	role, _ := GetRoleById(s.UID())
	if room, ok := RoomMgr.Rooms[role.roomId]; ok && room.Status < 10 {
		if role.IsReady() {
			Response(s, "fail")
		}
		// @todo 退出处理
		room.Group.Leave(s)
		switch room.Type {
		case 0: // 普通匹配
			if role.id == room.FPlayer {
				role2, _ := GetRoleById(room.SPlayer)
				role2.roomId = 0
				role2.status = 0
				room.Cast("onRoomDestroy", room.Type)
				delete(RoomMgr.Rooms, room.Id)
			} else if role.id == room.SPlayer {
				role2, _ := GetRoleById(room.FPlayer)
				role2.roomId = 0
				role2.status = 0

				room.Cast("onRoomDestroy", room.Type)
				delete(RoomMgr.Rooms, room.Id)
			}
			break
		case 1: // 好友赛
			var f int
			if role.id == room.FPlayer {

				f = 1
				room.FPlayer = 0
			} else if role.id == room.SPlayer {
				f = 2
				room.SPlayer = 0
			}

			if room.FPlayer == 0 && room.SPlayer == 0 {
				room.Cast("onRoomDestroy", room.Type)
				delete(RoomMgr.Rooms, room.Id)
			} else {
				room.Cast("onRoleQuit", f)
				room.Group.Leave(s)
			}
			break
		case 2: // 段位赛
			break
		}
		role.roomId = 0
		role.status = 0
		Response(s, "ok")
	} else {
		Response(s, "fail")
	}
	return nil
}

func (self *RoomHandlers) recover(s *session.Session, msg interface{}) {
	if err := recover(); err != nil {
		log.Println(err, msg)
		s.Push("notice", msg)
		Response(s, err)
	}
}
