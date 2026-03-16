package ws_main

import (
	"log"

	ws "golang.org/x/net/websocket"
)

type wsConn struct {
	conn   *ws.Conn
	sendCh chan []byte
}

func newWsConn(conn *ws.Conn) *wsConn {
	return &wsConn{
		conn:   conn,
		sendCh: make(chan []byte),
	}
}

func (conn *wsConn) readData() {
	for data := range conn.sendCh {
		if _, err := conn.conn.Write(data); err != nil {
			log.Printf("ERROR: send data to ws conn: %v", err)
		}
	}
}

func (conn *wsConn) close() {
	conn.conn.Close()
	close(conn.sendCh)
}
