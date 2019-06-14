package socket

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Token string
	ReaderChannel chan  string
	WriteHannel chan string
}


