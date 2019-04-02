package main

import (
	"time"

	"golang.org/x/gofrontend/libgo/go/log"
)

// 匹配
var matchSlice []int64

func MatchGro(mgr *RoomManager) {
	for {
		matchTime := time.After(2 * time.Second)
		select {
		case id := <-mgr.MatchChannel:
			matchSlice = append(matchSlice, id)
		case id := <-mgr.MatchCancelChannel:
			flag := true
			for index, v := range matchSlice {
				if v == id { // 取消成功
					matchSlice = append(matchSlice[:index], matchSlice[index+1:]...)
					role, ok := GetRoleById(id)
					if ok {
						role.Push("onCancelMatch", &CancelMatchRes{0})
						role.status = 0
					}
					flag = false
				}
			}
			// 	取消失败
			if flag {
				role, ok := GetRoleById(id)
				if ok {
					role.Push("onCancleMatch", &CancelMatchRes{0})
				}
			}
		case <-matchTime:
			doMatch(mgr, matchSlice)
			if len(matchSlice)%2 == 0 {
				matchSlice = []int64{}
			} else {
				matchSlice = matchSlice[len(matchSlice)-1:]
			}

		}
	}
}

// do match
func doMatch(mgr *RoomManager, ids []int64) {
	// 不足两人继续等待 @todo 长时间没有人加入匹配机器人
	if len(ids) < 2 {
		return
	}
	for i := 0; i < len(ids)-1; i += 2 {
		_, ok1 := GetRoleById(ids[i])
		_, ok2 := GetRoleById(ids[i+1])
		if !ok1 || !ok2 {
			if ok1 {
				ids = append(ids, ids[i])
			}
			if ok2 {
				ids = append(ids, ids[i+1])
			}
			continue
		}
		roomId := mgr.NewRoomId()
		room := NewRoom(roomId)
		room.FPlayer = ids[i]
		room.SPlayer = ids[i+1]
		mgr.Rooms[roomId] = room
		room.Init(0)
	}
}
