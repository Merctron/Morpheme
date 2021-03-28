package main

import "fmt"
import "rsc.io/quote"
import "log"
import "net/http"
import "github.com/gorilla/websocket"
// import "io/ioutil"
import "os/exec"
// import "sync"

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel
var mathlang  = make(chan MathLangRequest)   // mathlang  channel
var writer    = make(chan Message)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Define our message object
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// Object to process mathlang requests
type MathLangRequest struct {
	Email         string `json:"email"`
	Username      string `json:"username"`
	ScriptName    string `json:"scriptName"`
	ScriptContent string `json:"scriptContent"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
			log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
				log.Printf("error: %v", err)
				delete(clients, ws)
				break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages() {
	for {
			// Grab the next message from the broadcast channel
			msg := <-broadcast
			log.Println("Recieved msg:", msg.Message)
			if msg.Message[0] == '`' {
				log.Println("Detected math lang script", msg.Message[1:len(msg.Message) - 1])
				mathLangReq := MathLangRequest{
					Email: msg.Email,
					Username: msg.Username,
					ScriptName: "",
					ScriptContent: msg.Message[1:len(msg.Message) - 1],
				}
				mathlang <- mathLangReq
			} else {
				writer <- msg
			}
			
	}
}

func writeToClient() {
	for {
		msg := <-writer

		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
					log.Printf("error: %v", err)
					client.Close()
					delete(clients, client)
			}
		}
	}
}

func handleMathLangMessage() {
	for {
		req := <-mathlang
		log.Println("Recieved math lang request:", req.ScriptContent)
		mathLangCmd := exec.Command("C:\\Users\\Murtuza Kainan\\Code\\M-lisp\\mlisp.exe", "--single-use", req.ScriptContent)
		cmdOut, err := mathLangCmd.Output()
		if err != nil {
			log.Println("Recieved err:", err)
		    panic(err)
		}
		log.Println("Recieved output:", string(cmdOut))
		res := Message{
			Email: req.Email,
			Username: req.Username,
			Message: "Recieved math lang request:" + req.ScriptContent + ", Output: " + string(cmdOut),
		}
		writer <- res
	}
}

func main() {
    fmt.Println(quote.Go())
	// Create a simple file server
	http.Handle("/", http.FileServer(http.Dir("./public")))

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()
	go writeToClient()
	go handleMathLangMessage()

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
			log.Fatal("ListenAndServe: ", err)
	}
}