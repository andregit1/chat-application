package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

type User struct {
	Username string
	Password string
	Conn     *websocket.Conn
}

type ChatRoom struct {
	Name    string
	Clients map[string]*User
}

var (
	upgrader     = websocket.Upgrader{}
	chatRooms    = make(map[string]*ChatRoom)
	users        = make(map[string]string)   // Username -> Password
	chatHistory  = make(map[string][]string) // RoomName -> Messages
	mutex        sync.Mutex
	usersFile    = "users.json"
	chatFile     = "chat.json"
)

func main() {
	loadData()
	cleanUsersData() // Clean invalid entries from users map

	var rootCmd = &cobra.Command{Use: "chat-app"}
	rootCmd.AddCommand(serverCmd, clientCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cleanUsersData() {
	mutex.Lock()
	defer mutex.Unlock()

	for username := range users {
		if username == "" {
			delete(users, username)
		}
	}
	saveToFile(usersFile, users)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start chat server",
	Run: func(cmd *cobra.Command, args []string) {
		http.HandleFunc("/ws", handleConnections)
		log.Println("Starting server on :8080")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	},
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade: %v", err)
		return
	}
	defer conn.Close()

	var user User
	err = conn.ReadJSON(&user)
	if err != nil {
		log.Printf("Failed to read user: %v", err)
		return
	}

	user.Conn = conn
	if !authenticateUser(&user) {
		user.Conn.WriteJSON(map[string]string{"error": "Invalid username or password"})
		return
	}

	user.Conn.WriteJSON(map[string]string{"success": "Authenticated"})
	for {
		var msg map[string]string
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			deleteUserFromAllRooms(user.Username)
			break
		}

		handleMessage(user, msg)
	}
}

func authenticateUser(user *User) bool {
	if user.Username == "" || user.Password == "" {
		return false
	}

	mutex.Lock()
	defer mutex.Unlock()

	username := user.Username
	password := user.Password

	storedPassword, exists := users[username]
	if !exists {
		return false
	}

	if storedPassword == password {
		return true
	}
	return false
}

func handleMessage(user User, msg map[string]string) {
	action := msg["action"]
	switch action {
	case "create":
		roomName := msg["room"]
		createRoom(roomName, user)
	case "join":
		roomName := msg["room"]
		joinRoom(roomName, user)
	case "leave":
		roomName := msg["room"]
		leaveRoom(roomName, user)
	case "message":
		roomName := msg["room"]
		broadcastMessage(roomName, user.Username, msg["message"])
	case "dm":
		recipient := msg["recipient"]
		sendDirectMessage(user.Username, recipient, msg["message"])
	}
}

func createRoom(name string, user User) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := chatRooms[name]; exists {
		user.Conn.WriteJSON(map[string]string{"error": "Room already exists"})
		return
	}

	chatRooms[name] = &ChatRoom{
		Name:    name,
		Clients: make(map[string]*User),
	}
	user.Conn.WriteJSON(map[string]string{"info": "Room created"})
	joinRoom(name, user)
}

func joinRoom(name string, user User) {
	mutex.Lock()
	defer mutex.Unlock()

	room, exists := chatRooms[name]
	if !exists {
		user.Conn.WriteJSON(map[string]string{"error": "Room does not exist"})
		return
	}

	room.Clients[user.Username] = &user
	user.Conn.WriteJSON(map[string]string{"info": "Joined room"})
	broadcastMessage(name, "System", fmt.Sprintf("%s has joined the room", user.Username))
}

func leaveRoom(name string, user User) {
	mutex.Lock()
	defer mutex.Unlock()

	room, exists := chatRooms[name]
	if !exists {
		user.Conn.WriteJSON(map[string]string{"error": "Room does not exist"})
		return
	}

	delete(room.Clients, user.Username)
	user.Conn.WriteJSON(map[string]string{"info": "Left room"})
	broadcastMessage(name, "System", fmt.Sprintf("%s has left the room", user.Username))
}

func broadcastMessage(roomName, sender, message string) {
	mutex.Lock()
	defer mutex.Unlock()

	room, exists := chatRooms[roomName]
	if !exists {
		log.Printf("Room %s does not exist", roomName)
		return
	}

	chatHistory[roomName] = append(chatHistory[roomName], fmt.Sprintf("%s: %s", sender, message))
	saveData()

	for username, client := range room.Clients {
		var displayMsg string
		if username == sender {
			displayMsg = fmt.Sprintf("(you): %s", message)
		} else {
			displayMsg = fmt.Sprintf("(%s): %s", sender, message)
		}
		client.Conn.WriteJSON(map[string]string{"sender": sender, "message": displayMsg})
	}
}

func sendDirectMessage(sender, recipient, message string) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, room := range chatRooms {
		if user, exists := room.Clients[recipient]; exists {
			user.Conn.WriteJSON(map[string]string{"sender": sender, "message": message, "dm": "true"})
			return
		}
	}
}

func deleteUserFromAllRooms(username string) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, room := range chatRooms {
		delete(room.Clients, username)
	}
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start chat client",
	Run: func(cmd *cobra.Command, args []string) {
		for {
			prompt := promptui.Select{
				Label: "Select Action",
				Items: []string{"Register", "Login", "Quit"},
			}
			_, action, err := prompt.Run()
			if err != nil {
				fmt.Println("Failed to read action:", err)
				continue
			}

			switch action {
			case "Register":
				username := promptUser("Enter Username")
				password := promptUser("Enter Password")
				registerUser(username, password)
				fmt.Println("Registration successful, please login.")
			case "Login":
				username := promptUser("Enter Username")
				password := promptUser("Enter Password")

				conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
				if err != nil {
					log.Fatal("Failed to connect to server:", err)
				}

				err = conn.WriteJSON(map[string]string{"username": username, "password": password})
				if err != nil {
					log.Fatal("Failed to send username and password:", err)
				}

				var authResponse map[string]string
				err = conn.ReadJSON(&authResponse)
				if err != nil {
					log.Fatal("Failed to read auth response:", err)
				}

				if errMsg, ok := authResponse["error"]; ok {
					fmt.Println(errMsg)
					conn.Close()
					continue
				}

				messageChannel := make(chan map[string]string)
				go readMessages(conn, messageChannel)

				go func() {
					for msg := range messageChannel {
						displayMessage(msg)
					}
				}()

				chatMenu(conn)
			case "Quit":
				os.Exit(0)
			}
		}
	},
}

func readMessages(conn *websocket.Conn, messageChannel chan map[string]string) {
	defer close(messageChannel)
	for {
		var msg map[string]string
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Connection closed by server")
			} else {
				log.Printf("Error reading message: %v", err)
			}
			return
		}
		messageChannel <- msg
	}
}

func displayMessage(msg map[string]string) {
	if dm, ok := msg["dm"]; ok && dm == "true" {
		fmt.Printf("DM from %s: %s\n", msg["sender"], msg["message"])
	} else {
		fmt.Printf("%s: %s\n", msg["sender"], msg["message"])
	}
}

func chatMenu(conn *websocket.Conn) {
	for {
		prompt := promptui.Select{
			Label: "Select Action",
			Items: []string{"Create Room", "Join Room", "Leave Room", "Send Message", "Direct Message", "Quit"},
		}
		_, action, err := prompt.Run()
		if err != nil {
			fmt.Println("Failed to read action:", err)
			return
		}

		handleClientAction(action, conn)
	}
}

func handleClientAction(action string, conn *websocket.Conn) {
	switch action {
	case "Create Room":
		roomName := promptUser("Enter Room Name")
		conn.WriteJSON(map[string]string{"action": "create", "room": roomName})
		chatRoomMenu(conn, roomName)
	case "Join Room":
		roomName := promptUser("Enter Room Name")
		conn.WriteJSON(map[string]string{"action": "join", "room": roomName})
		chatRoomMenu(conn, roomName)
	case "Leave Room":
		roomName := promptUser("Enter Room Name")
		conn.WriteJSON(map[string]string{"action": "leave", "room": roomName})
	case "Send Message":
		roomName := promptUser("Enter Room Name")
		message := promptUser("Enter Message")
		conn.WriteJSON(map[string]string{"action": "message", "room": roomName, "message": message})
	case "Direct Message":
		recipient := promptUser("Enter Recipient Username")
		message := promptUser("Enter Message")
		conn.WriteJSON(map[string]string{"action": "dm", "recipient": recipient, "message": message})
	case "Quit":
		conn.Close()
		os.Exit(0)
	}
}

func chatRoomMenu(conn *websocket.Conn, roomName string) {
	for {
		prompt := promptui.Select{
			Label: roomName + " Menu",
			Items: []string{"Send Message", "Leave Room", "Back to Menu"},
		}
		_, action, err := prompt.Run()
		if err != nil {
			fmt.Println("Failed to read action:", err)
			return
		}

		switch action {
		case "Send Message":
			message := promptUser("Enter Message")
			conn.WriteJSON(map[string]string{"action": "message", "room": roomName, "message": message})
		case "Leave Room":
			conn.WriteJSON(map[string]string{"action": "leave", "room": roomName})
			return
		case "Back to Menu":
			return
		}
	}
}

func promptUser(label string) string {
	prompt := promptui.Prompt{
		Label: label,
	}
	result, err := prompt.Run()
	if err != nil {
		log.Fatal("Prompt failed:", err)
	}
	return result
}

func registerUser(username, password string) {
	if username == "" || password == "" {
		fmt.Println("Username and password cannot be empty")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	users[username] = password
	saveData()
}

func saveData() {
	saveToFile(usersFile, users)
	saveToFile(chatFile, chatHistory)
}

func loadData() {
	loadFromFile(usersFile, &users)
	loadFromFile(chatFile, &chatHistory)
}

func saveToFile(filename string, data interface{}) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file %s: %v", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		log.Fatalf("Failed to encode data to file %s: %v", filename, err)
	}
}

func loadFromFile(filename string, data interface{}) {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to open file %s: %v", filename, err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(data)
	if err != nil {
		log.Printf("Failed to decode data from file %s: %v", filename, err)
	}
}
