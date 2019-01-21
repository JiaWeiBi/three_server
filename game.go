

package main

import "fmt"

type (
	Point struct {
		X int
		Y int
	}
	Step struct {
		Type int
		Src Point
		Dest Point
	}
)
var Points []Point
func init(){
	Points = []Point{
		Point{-3,-3},Point{-3,0},Point{-3,3},
		Point{-2,-2},Point{-2,0},Point{-2,2},
		Point{-1,-1},Point{-1,0},Point{-1,1},
		Point{0,-3},Point{0,-2},Point{0,-1},Point{0,1},Point{0,2},Point{0,3},
		Point{1,-1},Point{1,0},Point{1,1},
		Point{2,-2},Point{2,0},Point{2,2},
		Point{3,-3},Point{3,0},Point{3,3},
	}
}
func main() {
	Map := make(map[Point]int)
	for _,p := range Points{
		Map[p] = 0
	}
	Flag := 0 // 0 为休息
	var FPlayer, SPlayer int64 = 1,2
	var FPlayerHelpers,SPlayerHelpers []int64
	var HotPoints []Point
	var

		a := Point{1,2}

	fmt.Println(Map)

	fmt.Println(Map[a])
}