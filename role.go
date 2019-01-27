package main

import "github.com/lonng/nano/session"

type(
	Role struct {
		id int64
		name string
		icon string // 头像
		level int // 等级
		gold int64 // 金币

		roomId int64 // 所在房间ID
		// 后端数据
		session *session.Session

	}

	CastRole struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
		// 头像
		Icon string `json:"icon"`
		// 等级
		Level int `json:"level"`
		// 金币
		Gold int64 `json:"gold"`
	}
)

// 组装广播角色信息
func FormatCastRole(role *Role) *CastRole{
	castRole := &CastRole{}
	castRole.Id = role.id
	castRole.Name = role.name
	castRole.Icon = role.icon
	castRole.Level = role.level
	castRole.Gold = role.gold
	return castRole
}
func FormatCastRoles(roles []*Role) []*CastRole{
	res := make([]*CastRole, len(roles))
	for i, role := range roles{
		res[i] = FormatCastRole(role)
	}
	return res
}