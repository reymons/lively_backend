package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"lively/config"
	"lively/core/service"
	"lively/db/pg"
	"lively/security/jwt"
	"lively/security/password"
	"lively/store"

	"lively/infra/mem_pubsub"

	"lively/transport/http"
	"lively/transport/media_channel"
	"lively/transport/rtmp"
	"lively/transport/ws_main"
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

	if err := transport.RunServer(addr, conf.HTTPAllowedOrigins); err != nil {
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

	// Infrastructure
	eventBus := mem_pubsub.NewBus()

	// Misc
	pswdManager := password.NewManager()

	jwtService := jwt.NewService(
		jwt.NewToken(conf.JWTAccessTokenSecret, time.Hour*24*30),
		jwt.NewToken("", 0),
	)

	// Core services
	authService := service.NewAuth(dbClient, users, streamKeys, pswdManager)
	skService := service.NewStreamKey(dbClient, streamKeys)
	streamService := service.NewStream(eventBus)

	// Transports
	mediaChannel := media_channel.New()
	rtmpTransport := rtmp.NewTransport(mediaChannel, skService, streamService)

	httpTransport := http.NewTransport(
		[]http.HandlerInfo{
			{"/ws/streams/", ws_media.NewTransport(mediaChannel, streamService)},
			{"/ws/main", ws_main.NewTransport(eventBus.NewSubscriber())},
		},

		&http.Dependencies{
			// Stores
			Users: users,
			// Core services
			AuthService:      authService,
			StreamKeyService: skService,
			// Misc
			JWTService: jwtService,
			DBClient:   dbClient,
		},
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go runRTMPServer(&wg, rtmpTransport, conf)
	go runHTTPServer(&wg, httpTransport, conf)
	wg.Wait()
}
