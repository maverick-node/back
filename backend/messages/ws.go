package messages

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"social-net/db"
	logger "social-net/log"
	"social-net/notification"
	"social-net/session"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

type Message struct {
	Message  string    `json:"message"`
	Username string    `json:"username"`
	Receiver string    `json:"receiver"`
	Time     time.Time `json:"time"`
	Type     string    `json:"type"`
}

var (
	onlineUsers  = make(map[string]bool)
	onlineMutex  sync.Mutex
	clientsMutex sync.Mutex

	clients = make(map[string][]*websocket.Conn)
	upgrade = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func Handleconnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {

		logger.LogError("Error upgrading connection", err)
		return
	}
	defer conn.Close()

	sessionn, err := r.Cookie("token")
	if err != nil {

		logger.LogError("Error getting token", err)
		return
	}
	token := sessionn.Value
	userid, ok := session.GetUserIDFromToken(token)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username, ok := session.GetUsernameFromUserID(userid)
	if !ok {
		http.Error(w, "Unauthorized: Invalid username", http.StatusUnauthorized)
		return
	}
	clientsMutex.Lock()

	clients[username] = append(clients[username], conn)
	onlineUsers[username] = true
	clientsMutex.Unlock()

	broadcastOnlineUsers()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			logger.LogError("Error reading message", err)
			break
		}

		sendMessageToRecipient(msg)
		notification.CreateNotificationMessage(msg.Receiver, msg.Username, "message", msg.Message)
		saveMessageToDB(msg.Username, msg.Receiver, msg.Message, msg.Type)
	}
	clientsMutex.Lock()
	conns := clients[username]
	for i, c := range conns {
		if c == conn {
			clients[username] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(clients[username]) == 0 {
		delete(onlineUsers, username)
	}
	clientsMutex.Unlock()

	log.Printf("User %s disconnected (remaining connections: %d)\n", username, len(clients[username]))
	broadcastOnlineUsers()
}

func saveMessageToDB(sender string, receiver string, message string, typee string) error {
	senderID, err := session.GetUserIDFromUsername(sender)
	if err != nil {
		logger.LogError("Failed to get sender ID", err)
		return fmt.Errorf("failed to get sender ID: %w", err)
	}

	receiverID, err := session.GetUserIDFromUsername(receiver)
	if err != nil {
		logger.LogError("Failed to get receiver ID", err)
		return fmt.Errorf("failed to get receiver ID: %w", err)
	}

	if senderID == receiverID {
		return fmt.Errorf("sender and receiver cannot be the same")
	}
	messageID, errr := uuid.NewV7()
	if errr != nil {
		return fmt.Errorf("failed to generate message ID: %w", errr)
	}
	if typee != "typing" {
		pre, err := db.DB.Prepare("INSERT INTO messages (id,sender_id, receiver_id, content, creation_date) VALUES (?,?, ?, ?, ?)")
		if err != nil {
			logger.LogError("Failed to prepare statement", err)
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer pre.Close()

		_, err = pre.Exec(messageID, senderID, receiverID, message, time.Now())
		if err != nil {
			logger.LogError("Failed to execute statement", err)
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}

func GetMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") //

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sessionn, err := r.Cookie("token")
	if err != nil {
		log.Println(err)
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	token := sessionn.Value
	userid, ok := session.GetUserIDFromToken(token)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username, ok := session.GetUsernameFromUserID(userid)
	if !ok {
		http.Error(w, "Unauthorized: Invalid username", http.StatusUnauthorized)
		return
	}

	sender := r.URL.Query().Get("sender")
	receiver := r.URL.Query().Get("receiver")

	if sender == "" || receiver == "" {
		http.Error(w, "Sender and receiver are required", http.StatusBadRequest)
		return
	}
	if sender != username && receiver != username {
		http.Error(w, "You are not authorized to view these messages", http.StatusForbidden)
		return
	}
	senderID, ok1 := session.GetUserIDFromUsername(sender)
	if ok1 != nil {
		http.Error(w, "Invalid sender username", http.StatusBadRequest)
		return
	}
	receiverID, ok2 := session.GetUserIDFromUsername(receiver)
	if ok2 != nil {
		http.Error(w, "Invalid receiver username", http.StatusBadRequest)
		return
	}
	if senderID == receiverID {
		http.Error(w, "Sender and receiver cannot be the same", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query(`
		SELECT sender_id, receiver_id, content, creation_date
		FROM messages
		WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
		ORDER BY creation_date ASC
	`, senderID, receiverID, receiverID, senderID)
	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		logger.LogError("Failed to fetch messages", err)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var senderID, receiverID string
		var content string
		var creationDate time.Time

		err := rows.Scan(&senderID, &receiverID, &content, &creationDate)
		if err != nil {
			http.Error(w, "Error scanning message row", http.StatusInternalServerError)
			logger.LogError("Error scanning message row", err)
			return
		}

		senderUsername, _ := session.GetUsernameFromUserID(senderID)
		receiverUsername, _ := session.GetUsernameFromUserID(receiverID)

		messages = append(messages, Message{
			Username: senderUsername,
			Message:  content,
			Receiver: receiverUsername,
			Time:     creationDate,
		})
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error iterating over rows", http.StatusInternalServerError)
		logger.LogError("Error iterating over rows", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		http.Error(w, "Failed to encode messages to JSON", http.StatusInternalServerError)
		logger.LogError("Failed to encode messages to JSON", err)
	}
}

func sendMessageToRecipient(msg Message) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	recipientConns, ok := clients[msg.Receiver]
	if !ok {
		log.Printf("Error: Recipient %s not connected\n", msg.Receiver)
		return
	}

	for _, conn := range recipientConns {
		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error sending message to %s: %v\n", msg.Receiver, err)
			logger.LogError("Error sending message", err)
		}
	}
}

func broadcastOnlineUsers() {
	onlineMutex.Lock()
	defer onlineMutex.Unlock()

	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for clientUsername, conns := range clients {

		allUsers, err := GetAllUsersExceptCurrent(clientUsername)
		if err != nil {
			log.Println("Error fetching all users:", err)
			logger.LogError("Error fetching all users", err)
			continue
		}

		var onlineUsersList []string
		var offlineUsersList []string

		for _, user := range allUsers {
			if onlineUsers[user] {
				onlineUsersList = append(onlineUsersList, user)
			} else {
				offlineUsersList = append(offlineUsersList, user)
			}
		}

		message := map[string]interface{}{
			"type":         "users",
			"onlineuser":   onlineUsersList,
			"offlineusers": offlineUsersList,
		}
		for _, conn := range conns {
			err := conn.WriteJSON(message)
			if err != nil {
				log.Printf("Error sending online/offline users list to %s: %v\n", clientUsername, err)
				logger.LogError("Error sending online/offline users list", err)
			}
		}
	}
}

func GetAllUsers() ([]string, error) {
	rows, err := db.DB.Query("SELECT username FROM users")
	if err != nil {
		log.Println("Error querying database:", err)
		logger.LogError("Error querying database", err)
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		err := rows.Scan(&username)
		if err != nil {
			log.Println("Error scanning row:", err)
			logger.LogError("Error scanning row", err)
			return nil, err
		}
		users = append(users, username)
	}

	if err := rows.Err(); err != nil {
		log.Println("Error with rows iteration:", err)
		logger.LogError("Error with rows iteration", err)
		return nil, err
	}

	return users, nil
}

func GetAllUsersExceptCurrent(currentUsername string) ([]string, error) {
	rows, err := db.DB.Query("SELECT username FROM users WHERE username != ?", currentUsername)
	if err != nil {
		log.Println("Error querying database:", err)
		logger.LogError("Error querying database", err)
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		err := rows.Scan(&username)
		if err != nil {
			log.Println("Error scanning row:", err)
			logger.LogError("Error scanning row", err)
			return nil, err
		}
		users = append(users, username)
	}

	if err := rows.Err(); err != nil {
		log.Println("Error with rows iteration:", err)
		logger.LogError("Error with rows iteration", err)
		return nil, err
	}

	return users, nil
}
