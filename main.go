package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	socketio "github.com/googollee/go-socket.io"
)

type Msg struct {
	RoomName       string            `json:"room_name"`
	Time           int64             `json:"time"`
	Content        string            `json:"content"`
	ClientId       string            `json:"client_id"`
	ClientName     string            `json:"client_name"`
	FromClientId   string            `json:"from_client_id"`
	FromClientName string            `json:"from_client_name"`
	ToClientId     string            `json:"to_client_id"`
	ToClientName   string            `json:"to_client_name"`
	ClientList     map[string]string `json:"client_list"`
}

type SocketData struct {
	RoomName   string `json:"room_name"`
	ClientName string `json:"client_name"`
}

func main() {
	router := gin.New()

	server := socketio.NewServer(nil)

	// 	//該使用者有哪幾間房
	// 	fmt.Println("socket.Rooms()", socket.Rooms())
	// 	//全部共有開幾間房
	// 	fmt.Println("server.Rooms()", server.Rooms("/"))
	// 	//某房間有幾個人
	// 	fmt.Println("server.RoomLen()", server.RoomLen("/", "bcast"))

	server.OnConnect("/", func(socket socketio.Conn) error {

		fmt.Println("connection socket id : ", socket.ID())

		return nil
	})

	server.OnEvent("/", "login", func(socket socketio.Conn, req string) {

		fmt.Println("login.req :  ", req)

		login := []byte(req)

		var msg Msg

		json.Unmarshal(login, &msg)

		msg.Time = time.Now().Unix()
		msg.ClientId = socket.ID()

		clientList := make(map[string]string)

		server.ForEach("/", msg.RoomName, func(clientSocket socketio.Conn) {
			clientList[clientSocket.ID()] = clientSocket.Context().(SocketData).ClientName
			clientSocket.Emit("login", msg)
		})

		context := SocketData{
			RoomName:   msg.RoomName,
			ClientName: msg.ClientName,
		}

		socket.SetContext(context)

		socket.Rooms()

		socket.Join(msg.RoomName)

		msg.ClientList = clientList

		socket.Emit("login", msg)
	})

	server.OnEvent("/", "say", func(socket socketio.Conn, req string) {

		fmt.Println("say.req :  ", req)

		say := []byte(req)

		var msg Msg

		json.Unmarshal(say, &msg)

		msg.Time = time.Now().Unix()
		msg.FromClientId = socket.ID()
		msg.FromClientName = socket.Context().(SocketData).ClientName
		msg.ClientId = socket.ID()
		msg.ClientName = socket.Context().(SocketData).ClientName

		origContent := msg.Content

		server.ForEach("/", socket.Context().(SocketData).RoomName, func(clientSocket socketio.Conn) {

			if msg.ToClientId != "all" {

				if msg.ToClientId == clientSocket.ID() {

					msg.Content = "<b>對你说: </b>" + origContent

				} else if msg.ClientId == clientSocket.ID() {

					msg.Content = "<b>你對" + msg.ToClientName + "说: </b>" + origContent

				} else {

					return

				}

			}

			clientSocket.Emit("say", msg)

		})

	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(socket socketio.Conn, reason string) {

		fmt.Println("disconnect socker id :   ", socket.ID())
		fmt.Println("disconnect reason :   ", reason)

		if socket.Context() == nil {
			return
		}

		roomUserCount := 0

		msg := Msg{
			Time:           time.Now().Unix(),
			FromClientId:   socket.ID(),
			FromClientName: socket.Context().(SocketData).ClientName,
		}

		server.ForEach("/", socket.Context().(SocketData).RoomName, func(clientSocket socketio.Conn) {

			if socket.ID() != clientSocket.ID() {

				clientSocket.Emit("logout", msg)

				roomUserCount++
			}

		})

		if roomUserCount == 0 {
			fmt.Println("room " + socket.Context().(SocketData).RoomName + " is not one")
		}
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))
	router.Static("/static/", "./static")
	router.GET("/", func(c *gin.Context) {
		log.Println(c.Request.URL)
		if c.Request.URL.Path != "/" {
			http.Error(c.Writer, "Not found", http.StatusNotFound)
			return
		}
		if c.Request.Method != "GET" {
			http.Error(c.Writer, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	if err := router.Run("localhost:8000"); err != nil {
		log.Fatal("failed run app: ", err)
	}
}
