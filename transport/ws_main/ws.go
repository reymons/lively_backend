package ws_main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	ws "golang.org/x/net/websocket"

	"lively/core/pubsub"
	"lively/transport/ws_main/message"
)

type connMap = map[*wsConn]struct{}
type topicMap = map[string]struct{}

type Transport struct {
	events     pubsub.Subscriber
	topicConns map[string]connMap
	connTopics map[*wsConn]topicMap
	mu         sync.RWMutex
}

func NewTransport(events pubsub.Subscriber) *Transport {
	t := &Transport{
		events:     events,
		topicConns: make(map[string]connMap),
		connTopics: make(map[*wsConn]topicMap),
	}
	t.init()
	return t
}

func (t *Transport) init() {
	go t.readEvents()
}

func (t *Transport) closeConn(conn *wsConn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for topic := range t.connTopics[conn] {
		conns := t.topicConns[topic]
		delete(conns, conn)
		if len(conns) == 0 {
			delete(t.topicConns, topic)
			t.events.Unsubscribe(topic)
		}
	}

	delete(t.connTopics, conn)
	conn.close()
}

func (t *Transport) onSubscribe(conn *wsConn, data []byte) error {
	var mesg message.Subscribe
	if err := json.Unmarshal(data, &mesg); err != nil {
		return err
	}
	t.subscribe(mesg.Topic, conn)
	return nil
}

func (t *Transport) onUnsubscribe(conn *wsConn, data []byte) error {
	var mesg message.Unsubscribe
	if err := json.Unmarshal(data, &mesg); err != nil {
		return err
	}
	t.unsubscribe(mesg.Topic, conn)
	return nil
}

func (t *Transport) onMessage(conn *wsConn, mesg *Message) error {
	var err error

	switch mesg.Type {
	case "subscribe":
		err = t.onSubscribe(conn, mesg.Data)
	case "unsubscribe":
		err = t.onUnsubscribe(conn, mesg.Data)
	}

	return err
}

func (t *Transport) onConn(org *ws.Conn) {
	conn := newWsConn(org)
	defer t.closeConn(conn)

	go conn.readData()

	for {
		var buf []byte
		if err := ws.Message.Receive(org, &buf); err != nil {
			log.Printf("INFO: connection closed: %s, %v", org.Request().RemoteAddr, err)
			return
		}

		var mesg Message
		if err := json.Unmarshal(buf, &mesg); err != nil {
			log.Printf("ERROR: unmarshal ws data: %v", err)
			continue
		}

		if err := t.onMessage(conn, &mesg); err != nil {
			log.Printf("ERROR: handle message: %v", err)
		}
	}
}

func (t *Transport) subscribe(topic string, conn *wsConn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	conns := t.topicConns[topic]
	if conns == nil {
		conns = make(connMap, 1)
		t.topicConns[topic] = conns
		t.events.Subscribe(topic)
	}
	conns[conn] = struct{}{}

	topics := t.connTopics[conn]
	if len(topics) == 0 {
		topics = make(topicMap, 1)
		t.connTopics[conn] = topics
	}
	topics[topic] = struct{}{}
}

func (t *Transport) unsubscribe(topic string, conn *wsConn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	conns := t.topicConns[topic]
	delete(conns, conn)
	if len(conns) == 0 {
		delete(t.topicConns, topic)
		t.events.Unsubscribe(topic)
	}

	topics := t.connTopics[conn]
	delete(topics, topic)
	if len(topics) == 0 {
		delete(t.connTopics, conn)
	}
}

func (t *Transport) getEncodedMessage(typ string, data any) ([]byte, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return []byte{}, fmt.Errorf("marshal payload: %w", err)
	}
	return json.Marshal(&Message{Type: typ, Data: payload})
}

func (t *Transport) readEvents() {
	for {
		ev, err := t.events.Read()
		if err != nil {
			if err == pubsub.ErrSubscriberClosed {
				return
			}
			log.Printf("ERROR: read event: %v", err)
			continue
		}

		// TODO: split topic by sections and send events for each section
		conns := t.topicConns[ev.Topic()]
		if len(conns) == 0 {
			continue
		}

		data, err := t.getEncodedMessage(ev.Topic(), ev)
		if err != nil {
			log.Printf("ERROR: get encoded message: %v", err)
			continue
		}

		for conn := range conns {
			conn.sendCh <- data
		}
	}
}

func (t *Transport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler := ws.Handler(t.onConn)
	handler.ServeHTTP(w, req)
}
