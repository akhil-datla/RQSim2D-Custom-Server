package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/pterm/pterm"
)

var clients = make(map[string][]*websocket.Conn) // connected clients
var broadcast = make(chan Message)               // broadcast channel
// Configure the upgrader
var upgrader = websocket.Upgrader{}

// Define our message object
type Message struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Websocket *websocket.Conn `json:"-"`
}

func main() {
	banner()
	//read port flag
	portPtr := flag.Int("port", 8080, "port to listen on")
	flag.Parse()
	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)
	// Start listening for incoming chat messages
	go handleMessages()
	// Start the server on localhost port and log any errors
	pterm.Info.Printf("http server started on port %d\n", *portPtr)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *portPtr), nil)
	if err != nil {
		pterm.Error.Println("ListenAndServe: ", err)
		os.Exit(1)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)

		// Register our new client
		var clientFound bool
		for _, client := range clients[msg.Code] {
			if client == ws {
				clientFound = true
				break
			}
		}

		if !clientFound {
			clients[msg.Code] = append(clients[msg.Code], ws)
			pterm.Info.Println("Client connected")
		}

		if err != nil {
			pterm.Error.Printf("error: %v", err)
			for i, client := range clients[msg.Code] {
				if client == ws {
					remove(clients[msg.Code], i)
				}
			}
			break
		}
		msg.Websocket = ws
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}

}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast

		clientsByCode := clients[msg.Code]
		for _, client := range clientsByCode {
			err := client.WriteJSON(msg)
			if err != nil {
				pterm.Error.Printf("error: %v", err)
				client.Close()
				for i, client := range clients[msg.Code] {
					if client == msg.Websocket {
						remove(clients[msg.Code], i)
					}
				}
			}
		}
	}
}

func remove(s []*websocket.Conn, i int) []*websocket.Conn {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

//banner for the program
func banner() {
	pterm.DefaultCenter.Print(pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).WithMargin(10).Sprint("RQSimulator2D Websockets Server"))
	pterm.Info.Println("(c)2021 by Akhil Datla and Coach Chan")

}
