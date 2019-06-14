package main

import (
	"net/http"
	"log"
	"github.com/gorilla/websocket"
	"fmt"
	"encoding/json"
	"github.com/rfyiamcool/syncmap"
	"math/rand"
	"strconv"
	"time"
)

var session syncmap.Map
var upgrader = &websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Node struct {
	Conn    *websocket.Conn
	Token   string

	ReadMsg chan []byte
}
//用户发送消息
type MessageRequest struct {
	//1:群聊
	Type        int `json:"type"`
	Message     string `json:"message"`
	TargetToken string `json:"target_token"`
	Token       string `json:"token"`
	Name        string `json:"name"`
}
//输出消息
type MessageResponse struct {
	Code    int `json:"code"`
	Message string `json:"message"`
	Name string `json:"name"`
	Headimg string `json:headimg`
	Time    string `json:"time"`
}




func readMessage(node *Node) {
	for {
		_, p, err := node.Conn.ReadMessage()
		if err != nil {
			session.Delete(node.Token)
			log.Printf("token %s :下线 还有在线：%d", node.Token,*session.Length())
			return
		} else
		{
			node.ReadMsg <- p
			log.Println(p)
		}

	}
}
//处理分发消息
func distributionMessage(message *MessageRequest) {
	///群聊
	if message.Type == 1 {
		log.Print("群聊")
		session.Range(func(key, value interface{}) bool {
			print(key)
			fmt.Print(value)
			node, ok := value.(*Node)
			print(ok)
			if ok {
				response := MessageResponse{
					Message: message.Message,
					Headimg: "https://images.budiaodanle.com//Content/images/defaultheadimg.jpg",
					Name:    message.Name,
					Time:    time.Now().Format("15:04:05"),
				}
				bytes, _ := json.Marshal(response)
				node.Conn.WriteMessage(websocket.TextMessage, bytes)
			}
			return true
		})
	}else if(message.Type==2){//单聊

	}
}
//给客户端发送消息
func writeMessage(node *Node) {
	for {
		select {
		case msg := <-node.ReadMsg:
			log.Println(string(msg))
			m := MessageRequest{}
			err := json.Unmarshal(msg, &m)
			if err != nil {
				errMsg:=MessageResponse{
					Message:"消息格式不对:"+err.Error(),
					Code:-1,
				}
				bytes, _ := json.Marshal(errMsg)
				node.Conn.WriteMessage(websocket.TextMessage, bytes)
			} else {
				distributionMessage(&m)
			}

		}

	}
}

func main() {
	print(strconv.Itoa(rand.Int()))
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("ws")
		conn, err := upgrader.Upgrade(writer, request, nil)
		token := request.URL.Query().Get("token")
		node := &Node{
			Conn:    conn,
			Token:   token,
			ReadMsg: make(chan []byte, 1024),
		}
		session.Store(token, node)
		log.Printf("token %s :上线 还有在线：%d", node.Token,*session.Length())
		go readMessage(node)
		go writeMessage(node)

		if err != nil {
			log.Println(err.Error())
			return
		}
		writer.Write([]byte("链接成功"))

	})
	log.Println("服务开启：http://192.168.10.7:5600")
	log.Fatal(http.ListenAndServe(":5600", nil))
}
