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
	"net"
	"flag"
	"strings"
)

var session syncmap.Map
var udp *net.UDPConn
var ip *string
var upgrader = &websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Node struct {
	Conn  *websocket.Conn
	Token string

	ReadMsg chan []byte
}

//用户发送消息
type MessageRequest struct {
	//1:群聊
	Type        int    `json:"type"`
	Message     string `json:"message"`
	TargetToken string `json:"target_token"`
	Token       string `json:"token"`
	Name        string `json:"name"`
}

//输出消息
type MessageResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Name    string `json:"name"`
	Headimg string `json:headimg`
	Time    string `json:"time"`
}

func createUdp(){
	arr:=strings.Split(*ip,".")
	ip1, _ := strconv.Atoi(arr[0])
	ip2, _ := strconv.Atoi(arr[1])
	ip3, _ := strconv.Atoi(arr[2])
	ip4, _ := strconv.Atoi(arr[3])



	addr := net.UDPAddr{
		IP:net.IPv4(byte(ip1),byte(ip2),byte(ip3),byte(ip4)),
		Port:2100,
	}
	//广播地址
	raddr:=net.UDPAddr{
		IP:net.IPv4(255,255,255,255),
		Port:2100,
	}
	conn, e := net.DialUDP("udp", &addr, &raddr)
	if e!=nil{
		log.Printf(e.Error())
		return
	}
	udp=conn;

}

//通过UDP广播消息给所有机器
func udpBroadcast(node *Node) {

	for{
		select {
			case msg:=<-node.ReadMsg:

				udp.Write(msg)

		}
	}


}
//udp 收消息
func udpReceive()  {
	addr, e := net.ResolveUDPAddr("udp", ":2100")
	if e!=nil{
		log.Printf(e.Error())
		return
	}
	conn, error := net.ListenUDP("udp", addr)
	if error!=nil{
		log.Printf(error.Error())
		return
	}
	defer conn.Close()
	for{
		msg:=make([]byte,1024)
		n, _, error := conn.ReadFromUDP(msg)
		if error!=nil{
			log.Printf(error.Error())

		}else{
			log.Printf("udp收到消息:%s",string(msg))
			log.Println(string(msg))
			m := MessageRequest{}
			error:=json.Unmarshal(msg[:n], &m)
			if error!=nil{
				log.Printf("error ",error.Error())

			}else{
				distributionMessage(&m)
			}
		}
	}

}

func readMessage(node *Node) {
	for {
		_, p, err := node.Conn.ReadMessage()
		if err != nil {
			session.Delete(node.Token)
			log.Printf("token %s :下线 还有在线：%d", node.Token, *session.Length())
			return
		} else {
			//如果消息格式不对，就不传递
			go func(){
				m := MessageRequest{}
				err := json.Unmarshal(p, &m)
				if err != nil {
					errMsg := MessageResponse{
						Message: "消息格式不对:" + err.Error(),
						Code:    -1,
					}
					bytes, _ := json.Marshal(errMsg)
					node.Conn.WriteMessage(websocket.TextMessage,bytes)
				}else{
					node.ReadMsg <- p
					log.Println(p)
				}

			}()

		}

	}
}

//处理分发消息
func distributionMessage(message *MessageRequest) {
	///群聊
	print("11:",message.Type)
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
	} else if (message.Type == 2) { //单聊

	}
}

//给客户端发送消息
func writeMessage(node *Node,msg []byte) {
			log.Println(string(msg))
			m := MessageRequest{}
			err := json.Unmarshal(msg, &m)
			if err != nil {
				errMsg := MessageResponse{
					Message: "消息格式不对:" + err.Error(),
					Code:    -1,
				}
				bytes, _ := json.Marshal(errMsg)
				node.Conn.WriteMessage(websocket.TextMessage, bytes)
			} else {
				distributionMessage(&m)
			}




}

func main() {
	ip=flag.String("ip","192.168.10.7","ip address")
	flag.Parse()
	fmt.Print("ip",*ip)
	go createUdp()
	go udpReceive()
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
		log.Printf("token %s :上线 还有在线：%d", node.Token, *session.Length())
		go readMessage(node)
		//先通过UDP广播
		go udpBroadcast(node)

		if err != nil {
			log.Println(err.Error())
			return
		}
		writer.Write([]byte("链接成功"))

	})
	log.Println("服务开启：http://192.168.10.7:5600")
	log.Fatal(http.ListenAndServe(":5600", nil))
}
