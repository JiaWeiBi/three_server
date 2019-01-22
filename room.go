

package main

import (
	"github.com/lonng/nano"
	"github.com/lonng/nano/session"
)

type (
	Room struct {
		Group *nano.Group
		Timer *nano.Timer
		Woner int64
		// 房间状态 0:休息 1:下子 2:走子
		Status uint8
		// 棋盘表 0:为空 1：FPlayer棋子 2：SPlayer棋子 3：揪过棋子
		PointMap map[Point]int
		//0:休息 11:FPlayer下子 12:FPlayer走子 13:FPlayer揪子 21:SPlayer下子 22:SPlayer走子 23:SPlayer揪子
		Flag int
		FPlayer, SPlayer int64
		FPlayerHelpers,SPlayerHelpers []int64
		//
		//HotPoints []*Point
		//
		StepList []*Step
	}

	Point struct {
		X int
		Y int
	}
	Step struct {
		// 1:下子 2:走子 3:揪子
		Type int
		Src *Point
		Dest *Point
	}
)
var Points []Point
func init(){
	Points = []Point{
		{-3,-3},{-3,0},{-3,3},
		{-2,-2},{-2,0},{-2,2},
		{-1,-1},{-1,0},{-1,1},
		{0,-3},{0,-2},{0,-1},{0,1},{0,2},{0,3},
		{1,-1},{1,0},{1,1},
		{2,-2},{2,0},{2,2},
		{3,-3},{3,0},{3,3},
	}
}
func init() {
	Map := make(map[Point]int)
	for _,p := range Points{
		Map[p] = 0
	}
}

func NewRoom(woner int64) *Room{
	room := &Room{}
	room.Status = 0
	room.Flag = 0
	room.FPlayer = woner
	room.Woner = woner
	room.PointMap = make(map[Point]int)
	for _,p := range Points{
		room.PointMap[p] = 0
	}
	return room
}

func (* Room)Join(s *session.Session) (bool, string){

}