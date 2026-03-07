package main

import (
	"fmt"
	"log"
	"sync"

	"lively/config"
	"lively/core"
	"lively/db/pg"
	"lively/store"
	"lively/transport/rtmp"
)

func runRTMPServer(wg *sync.WaitGroup, rtmp *rtmp.Transport, conf *config.Config) {
	defer wg.Done()

	addr := fmt.Sprintf("%s:%s", conf.RTMPServerHost, conf.RTMPServerPort)
	log.Printf("INFO: running RTMP server on %s", addr)

	if err := rtmp.RunServer(addr, nil); err != nil {
		log.Fatalf("ERROR: run RTMP server: %v", err)
	}
}

func main() {
	conf := config.NewConfig()

	dbClient, err := pg.NewClient(conf.DatabaseURL)
	if err != nil {
		log.Fatalf("ERROR: new db client: %v", err)
	}
	defer dbClient.Close()

	// Stores
	users := store.NewUsers()
	streamKeys := store.NewStreamKeys()
	_ = users

	// Misc
	channel := core.NewMediaChannel()

	// Transports
	var wg sync.WaitGroup
	wg.Add(2)

	rtmp := rtmp.NewTransport(channel, dbClient, streamKeys)
	go runRTMPServer(&wg, rtmp, conf)

	wg.Wait()
}
