package main

import (
	"fmt"
	"log"
	"sync"

	"lively/config"
	"lively/core"
	"lively/db/pg"
	"lively/store"
	"lively/transport/http"
	"lively/transport/rtmp"
	"lively/transport/ws_media"
)

func runRTMPServer(wg *sync.WaitGroup, transport *rtmp.Transport, conf *config.Config) {
	defer wg.Done()

	addr := fmt.Sprintf("%s:%s", conf.RTMPServerHost, conf.RTMPServerPort)
	log.Printf("INFO: running RTMP server on %s", addr)

	if err := transport.RunServer(addr, nil); err != nil {
		log.Printf("ERROR: run RTMP server: %v", err)
	}
}

func runHTTPServer(wg *sync.WaitGroup, transport *http.Transport, conf *config.Config) {
	defer wg.Done()

	addr := fmt.Sprintf("%s:%s", conf.HTTPServerHost, conf.HTTPServerPort)
	log.Printf("INFO: running HTTP server on %s", addr)

	if err := transport.RunServer(addr); err != nil {
		log.Printf("ERROR: run HTTP server: %v", err)
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

	rtmpTransport := rtmp.NewTransport(channel, dbClient, streamKeys)

	handlers := []http.HandlerInfo{{"/ws/streams/", ws_media.NewTransport(channel)}}
	httpTransport := http.NewTransport(handlers)

	go runRTMPServer(&wg, rtmpTransport, conf)
	go runHTTPServer(&wg, httpTransport, conf)

	wg.Wait()
}
