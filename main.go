package main

import (
	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/serialize/json"
	"log"
	"net/http"
)

var RoomMgr *RoomManager
func main() {
	defer func() {
		DBExit()
	}()
	// rewrite component and handler Name
	roomMgr := NewRoomManager()
	nano.Register(roomMgr,
		component.WithName("roomMgr"),
	)
	roomHandlers := &RoomHandlers{}
	nano.Register(roomHandlers,
		component.WithName("roomHandlers"),
		)
	// override default serializer
	nano.SetSerializer(json.NewSerializer())
	// Init
	RoomMgr = roomMgr

	// 开启匹配协程
	go MatchGro(roomMgr)
	// traffic stats
	pipeline := nano.NewPipeline()
	var stats = &stats{}
	pipeline.Outbound().PushBack(stats.outbound)
	pipeline.Inbound().PushBack(stats.inbound)

	nano.EnableDebug()
	log.SetFlags(log.LstdFlags | log.Llongfile)
	nano.SetWSPath("/three_game")

	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	//nano.SetCheckOriginFunc(func(_ *http.Request) bool { return true })
	nano.ListenWS(":5000", nano.WithPipeline(pipeline))
}
