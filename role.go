package main

import (
	"github.com/lonng/nano"
	"github.com/lonng/nano/session"
	"golang.org/x/gofrontend/libgo/go/database/sql"
	"golang.org/x/gofrontend/libgo/go/log"
	"time"
)

type (
	Role struct {
		id     int64
		name   string
		icon   string // 头像
		level  int    // 等级
		score  int    // 分数
		gold   int64  // 金币
		gender uint8  //性别 0：未知、1：男、2：女

		roomId int64 // 所在房间ID
		// 后端数据
		// 状态 0:原始状态 1:匹配中
		status int
		session *session.Session
		openid  string
		// 清除定时器
		cleanTimer *nano.Timer
	}

	CastRole struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
		// 头像
		Icon string `json:"icon"`
		// 金币
		Gold int64 `json:"gold"`
		// 等级
		Level int `json:"level"`
		// 积分
		Score int `json:"score"`
		// 性别
		Gender uint8 `json:"gender"`
	}

)

//role
//玩家退出处理
func (role *Role)exit(){
	// 退出所在房间
	if _, ok := RoomMgr.Rooms[role.roomId];ok {
		role.QuitRoom()
	}
}

// 退出房间
func (role *Role)QuitRoom() bool{
	var f int
	if room, ok := RoomMgr.Rooms[role.roomId];ok {
		if room.Status > 10 && (role.id == room.SPlayer || role.id == room.FPlayer){
			// 结算，退出当认输处理
			var winner int
			if room.FPlayer == role.id{
				winner = 2
				f = 1
			}else{
				winner = 1
				f = 2
			}
			room.Settle(winner)
			room.Group.Leave(role.session)
			room.Group.Broadcast("onRoleQuit", f)
			return true
		}else if (room.Status == 1 && room.FPlayer == role.id) || (room.Status == 2 && room.SPlayer == role.id){
			role.session.Push("notice", "请先取消准备")
			return false
		}
	}
	role.roomId = 0
	role.status = 0
	return false
}

// 监测是否准备
func (role *Role)IsReady() bool{
	if room, ok := RoomMgr.Rooms[role.roomId];ok {
		if (room.Status == 1 && room.FPlayer == role.id) || (room.Status == 2 && room.SPlayer == role.id){
			return true
		}
	}
	return false
}

// 加金币
func (role *Role)AddGold(num int) bool{
	role.gold = role.gold + int64(num)
	_,err := StmpMap["updateGold"].Exec(role.gold,role.id)
	if err != nil{
		log.Panicln("添加金币失败,id:",role.id,",num:", num)
		log.Panicln(err)
		return false
	}
	role.session.Push("onGold", num)
	return true
}

// 加积分
func (role *Role)AddScore(num int){
	oldLevel := role.level
	role.score = role.score + num
	if role.score <= 0{
		role.score = 0
		role.level = 1
	}else{
		if role.score % 100 == 0{
			role.level = role.score / 100
		}else{
			role.level = role.score / 100 + 1
		}

	}
	_,err := StmpMap["updateScore"].Exec(role.score,role.level,role.id)
	if err != nil{
		log.Panicln("添加金币失败,id:",role.id,",num:", num)
		log.Panicln(err)
		return
	}
	role.session.Push("onScore", num)
	if oldLevel != role.level{
		role.session.Push("onLevel", role.level)
	}
	return
}

// 组装广播角色信息
func FormatCastRole(role *Role) *CastRole {
	castRole := &CastRole{}
	castRole.Id = role.id
	castRole.Name = role.name
	castRole.Icon = role.icon
	castRole.Level = role.level
	castRole.Score = role.score
	castRole.Gold = role.gold
	castRole.Gender = role.gender
	return castRole
}
func FormatCastRoles(roles []*Role) []*CastRole {
	res := make([]*CastRole, len(roles))
	for i, role := range roles {
		res[i] = FormatCastRole(role)
	}
	return res
}

// 通过openid查询mysql获取用户信息
func GetUserInfoByOpenid(openid string) (bool, *Role) {
	role := &Role{}
	err := mysqlConn.QueryRow("select id,openid,level,score,name,avatarUrl,gender,gold from threeUserInfo where openid=?", openid).Scan(
		&role.id, &role.openid, &role.level, &role.score, &role.name, &role.icon, &role.gender, &role.gold)
	if err != nil {
		if err.Error() == sql.ErrNoRows.Error() {
			return false, nil
		} else {
			panic(err)
		}
	} else {
		return true, role
	}
}

// 添加新用户
func AddNewUser(openid string, unionid string, userInfo *UserInfo) (*Role, error) {

	// 创建游戏账户
	now := time.Now().Unix()
	_, err := mysqlConn.Exec("insert INTO threeUserInfo(openid,level,score,createTime,name,avatarUrl,gender,province,city,gold) values(?,?,?,?,?,?,?,?,?,?)",
		openid, 1, 100, now, userInfo.NickName, userInfo.AvatarUrl, userInfo.Gender, userInfo.Province, userInfo.City, 100)
	if err != nil {
		log.Printf("threeUserInfo Insert failed,err:%v", err)
		return nil, err
	}

	if err != nil {
		log.Printf("threeUserInfo Get lastInsertID failed,err:%v", err)
		return nil, err
	}
	_, role := GetUserInfoByOpenid(openid)
	return role, nil
}
