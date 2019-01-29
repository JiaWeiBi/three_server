package main

import (
	"fmt"
	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
	"log"
	"time"
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
		ChessNum int // 总共的下子数量，揪子不减少
		// 超时次数 连续第三次超时判定为输
		FTimeOutTime int
		STimeOutTime int
	}

	// 房间信息 用于推送
	CastRoomInfo struct {
		Woner         uint8          `json:"woner"`
		Status        uint8          `json:"status"`
		PointMap      map[string]int `json:"pointmap"`
		Flag          int            `json:"flag"`
		FPlayer       *CastRole      `json:"fplayer"`
		SPlayer       *CastRole      `json:"splayer"`
		FPlayerHelper *CastRole      `json:"fhelper"`
		SPlayerHelper *CastRole      `json:"shelper"`
	}

	Point struct {
		X int `json:"X"`
		Y int `json:"Y"`
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
		Winner int `json:"winner"`
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
	fmt.Println("Init======")
	fmt.Println(C3CheckMap)
	borderFun := func(p1 *Point, p2 *Point)bool{
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
			if p2.Y == 0 && (p1.Y-p2.Y == 1 || p1.Y-p2.Y == -1) {
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
	for _,p1 := range Points{
		for _,p2 := range Points{
			if borderFun(&p1,&p2){
				BorderPointMap[p1] = append(BorderPointMap[p1], p2)
			}
		}
	}
	fmt.Println(BorderPointMap)
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

func (p *Point) ToString() string {
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
	for p, v := range room.PointMap {
		info.PointMap[p.ToString()] = v
	}
	return info
}

// 开始 @todo 重置数据
func (room *Room) Start() {
	// 数据初始化

	// 设置超时计时器
	room.Timer = nano.NewTimer(10*time.Second, room.TimeOut)
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
		if step.Dst.Y == 0 && (step.Src.Y-step.Dst.Y == 1 || step.Src.Y-step.Dst.Y == -1) {
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
	if room.Status < 10 || room.Status % 10 != 2{
		return false
	}
	Winner := 0
	var l1,l2 []Point
	for _, p := range Points {
		if room.PointMap[p] == 1 {
			l1 = append(l1, p)
		} else if room.PointMap[p] == 2 {
			l2 = append(l2, p)
		}
	}

	switch room.Status {
	case 12:
		if len(l1) < 3{
			Winner = 2
		}else{
			flag := true
			for _,p := range l1{
				for _, bp := range BorderPointMap[p]{
					if room.PointMap[bp] == 0 || room.PointMap[bp] == 3{
						flag = false
						break
					}
				}
				if !flag{break}
			}
			if flag{ Winner = 2}
		}

		break
	case 22:
		if len(l2) < 3{
			Winner = 1
		}else{
			flag := true
			for _,p := range l2{
				for _, bp := range BorderPointMap[p]{
					if room.PointMap[bp] == 0 || room.PointMap[bp] == 3{
						flag = false
						break
					}
				}
				if !flag{break}
			}
			if flag{ Winner = 2}
		}
		break
	}
	// @todo 输赢处理
	switch Winner {
	case 1:
		break
	case 2:
		break
	default:
		return false
	}
	settle := SettleMsg{Winner:Winner}
	room.Group.Broadcast("onSettle", settle)
	return true
}

//@todo  超时处理
func (room *Room) TimeOut() {

}

// 房间状态修改并广播
func (room *Room) ChangeStatus(s uint8) {
	room.Status = s
	room.Group.Broadcast("onStatus", room.Status)
	if room.Status%10 == 2{
		room.CheckWin()
	}
}

// 广播步骤
func (room *Room) CastStep(step *Step) {
	err := room.Group.Broadcast("onStep", step)
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
			room.Status = 1
		} else if room.SPlayer == uid {
			room.Status = 2
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
		s.Response("fail")
		return nil
	}
	s.Response("ok")
	return nil
}
func (self *RoomHandlers) CancleReady(s *session.Session, msg []byte) error {
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
			s.Response("ok")
		}
		return nil
	case 2:
		if room.SPlayer == uid {
			room.ChangeStatus(0)
			s.Response("ok")
		}
		return nil
	}
	s.Response("fail")
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
			s.Response("fail")
			return nil
		}
		if room.PointMap[*p] != 0 {
			s.Push("notice", NoticeMessage{Content: "当前位置不能落子"})
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
		s.Response("ok")
		break
	case 21:
		// 二号玩家落子
		if room.SPlayer != uid {
			s.Response("fail")
			return nil
		}
		if room.PointMap[*p] != 0 {
			s.Push("notice", NoticeMessage{Content: "当前位置不能落子"})
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
		s.Response("ok")
		break
	default:
		s.Response("fail")
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
			s.Response("fail")
			return nil
		}
		if !room.CheckTake(p) {
			s.Push("notice", NoticeMessage{Content: "此位置不能揪子"})
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
		s.Response("ok")
		break
	case 23:
		if room.SPlayer != uid {
			s.Response("fail")
			return nil
		}
		if !room.CheckTake(p) {
			s.Push("notice", NoticeMessage{Content: "此位置不能揪子"})
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
		s.Response("ok")
		break
	}
	return nil
}

// 走子
func (self *RoomHandlers) Move(s *session.Session, step *Step) error {
	fmt.Println(step.Src, "=====", step.Dst)
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
		s.Response("ok")
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
		s.Response("ok")
		break
	default:
		s.Response("fail")
		return nil
	}
	// 广播
	room.StepList = append(room.StepList, step)
	room.CastStep(step)
	return nil
}

func (self *RoomHandlers) recover(s *session.Session, msg interface{}) {
	if err := recover(); err != nil {
		log.Println(err, msg)
		s.Response(err)
	}
}
