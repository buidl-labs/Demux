package main

import (
	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/server"
)

func main() {
	dataservice.InitDB()
	server.StartServer(":8000")
}
