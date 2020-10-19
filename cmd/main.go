package main

import (
	"os"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/server"
	"github.com/buidl-labs/Demux/util"
)

func main() {
	dataservice.InitMongoClient()
	go util.RunPoller()
	server.StartServer(":" + os.Getenv("PORT"))
}
