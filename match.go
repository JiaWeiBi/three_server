package main

import (
	"fmt"
	"time"
)

// 匹配
func MatchGro(mgr *RoomManager){
	matchSlice := []int64{}
	for{
		matchTime := time.After(2 * time.Second)
		select {
		case id := <- mgr.MatchChannel:
			fmt.Println("====match Id:", id)
			matchSlice = append(matchSlice, id)
		case <- matchTime:
			fmt.Println("====matchTime")
			doMatch(mgr, matchSlice)
			if len(matchSlice) % 2 == 0{
				matchSlice = []int64{}
			}else{
				matchSlice = matchSlice[len(matchSlice)-1:]
			}

		}
	}
}
// do match
func doMatch(mgr *RoomManager, ids []int64){
	// 不足两人继续等待 @todo 长时间没有人加入匹配机器人
	if len(ids) < 2{
		return
	}
	for i:=0;i<len(ids)-1;i += 2{
		roomId := mgr.NewRoomId()
		room := NewRoom(roomId)
		room.FPlayer = ids[i]
		room.SPlayer = ids[i+1]
		mgr.Rooms[roomId] = room

		room.Init()
	}
}
