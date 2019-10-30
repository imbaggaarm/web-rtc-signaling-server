package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

//mockup data for users name and password
var user_accounts = make(map[string]string)

var users = make(map[string]*websocket.Conn)

var upgrader = websocket.Upgrader{
	//ReadBufferSize: 1024,
	//WriteBufferSize: 1024,
	//
	//CheckOrigin: func(r *http.Request) bool {
	//	return true
	//},
}

type Message struct {
	Type string `json:"type"`
	Data Data   `json:"data"`
}

type Data struct {
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`

	Username  string      `json:"username"`
	Candidate interface{} `json:"candidate"`
	Offer     interface{} `json:"offer"`
	Answer    interface{} `json:"answer"`
	Success   bool        `json:"success"`
}

type OfferMessage struct {
	FromID string      `json:"from_id"`
	ToID   string      `json:"to_id"`
	Offer  interface{} `json:"offer"`
}

//
//type OfferResponse struct {
//	FromID string `json:"from_id"`
//	Success bool `json:"success"`
//}

type LoginResponse struct {
	Type    string `json:"type"`
	Success bool   `json:"success"`
}

func main() {
	fmt.Println("Signaling server")

	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	//create mock data with username and password
	user_accounts["user1"] = "123456"
	user_accounts["user2"] = "123456"
	user_accounts["user3"] = "123456"

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("connected from: %s", ws.RemoteAddr())
	// Make sure we close the connection when the function returns
	defer ws.Close()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			for userID, conn := range users {
				if conn == ws {
					delete(users, userID)
					break
				}
			}
			break
		}
		log.Println("")
		log.Println("Message received type:", msg.Type)
		switch msg.Type {
		case "LOGIN":
			log.Println(msg.Type)
			log.Println(msg.Data.Username)
			//save user connection on the server
			users[msg.Data.Username] = ws

			//return login result
			err := ws.WriteJSON(Message{
				Type: "LOGIN_RESPONSE",
				Data: Data{
					Success: true,
				},
			})
			if err != nil {
				log.Println("Write error:", err)
			}
		case "OFFER":
			log.Println(msg.Type)
			log.Println(msg.Data.FromID)
			log.Println(msg.Data.ToID)
			//log.Println(msg.Data.Offer)
			conn := users[msg.Data.ToID]
			if conn != nil {
				log.Println("Sending offer to:", msg.Data.ToID)
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Write error:", err)
				}
			} else {
				log.Println("User", msg.Data.ToID, "not online")
				err := ws.WriteJSON(Message{
					Type: "OFFER_RESPONSE",
					Data: Data{
						FromID:  msg.Data.ToID,
						Success: false,
					},
				})
				if err != nil {
					log.Println("Write error:", err)
				}
			}
			//log.Println(msg.Data.Offer)
		case "ANSWER":
			log.Println(msg.Type)
			log.Println(msg.Data.FromID)
			log.Println(msg.Data.ToID)
			log.Println(msg.Data.Answer)

			conn := users[msg.Data.ToID]
			if conn != nil {
				log.Println("Sending answer to:", msg.Data.ToID)
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Write error:", err)
				}
			}
		case "CANDIDATE":
			log.Println(msg.Type)
			log.Println(msg.Data.FromID)
			log.Println(msg.Data.ToID)
			log.Println(msg.Data.Candidate)

			//handle send candidate to user
			conn := users[msg.Data.ToID]
			if conn != nil {
				log.Println("Sending candidate to:", msg.Data.ToID)
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Write error:", err)
				}
			}
		case "LEAVE":
			log.Println(msg.Type)
			log.Println(msg.Data.FromID)
			log.Println(msg.Data.ToID)
		default:
			log.Println("Error: Unexpected type: ", msg.Type)
		}
	}
}
