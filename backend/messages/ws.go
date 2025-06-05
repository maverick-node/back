package messages

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"social-net/db"
	logger "social-net/log"
	"social-net/notification"
	"social-net/session"
	"sync"
	"time"

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
	log.Printf("[WebSocket] New connection attempt from %s", r.RemoteAddr)

	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		logger.LogError("[WebSocket] Error upgrading connection", err)
		return
	}
	defer conn.Close()
	log.Printf("[WebSocket] Connection upgraded successfully")

	// Register client
	sessionn, err := r.Cookie("token")
	if err != nil {
		logger.LogError("[WebSocket] Error getting token", err)
		return
	}
	token := sessionn.Value
	userid, ok := session.GetUserIDFromToken(token)
	if !ok || userid == "" {
		log.Printf("[WebSocket] Invalid token or empty userid")
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username, ok := session.GetUsernameFromUserID(userid)
	if !ok {
		log.Printf("[WebSocket] Invalid username for userid: %s", userid)
		http.Error(w, "Unauthorized: Invalid username", http.StatusUnauthorized)
		return
	}
	log.Printf("[WebSocket] User authenticated: %s (ID: %s)", username, userid)

	clientsMutex.Lock()
	clients[username] = append(clients[username], conn) // Append new connection
	onlineUsers[username] = true
	clientsMutex.Unlock()
	log.Printf("[WebSocket] User %s registered with %d total connections", username, len(clients[username]))

	broadcastOnlineUsers()
	log.Printf("[WebSocket] Starting message loop for user: %s", username)

	// Send welcome message
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Error reading message from %s: %v", username, err)
			}
			break
		}
		fmt.Printf("[WebSocket] Received message from %s: %+v", username, msg)
		// Log the raw message structure
		msgJSON, _ := json.Marshal(msg)
		log.Printf("[WebSocket] Raw message received: %s", string(msgJSON))

		// Validate message structure
		if msg.Receiver == "" {
			log.Printf("[WebSocket] Error: Empty receiver in message from %s. Message structure: %+v", username, msg)
			// Send error back to client
			errorMsg := map[string]string{
				"error": "Receiver is required",
				"type":  "error",
			}
			conn.WriteJSON(errorMsg)
			continue
		}
		if msg.Message == "" {
			log.Printf("[WebSocket] Error: Empty message content from %s", username)
			errorMsg := map[string]string{
				"error": "Message content is required",
				"type":  "error",
			}
			conn.WriteJSON(errorMsg)
			continue
		}
		if msg.Type == "" {
			msg.Type = "message" // Set default type if not provided
		}

		log.Printf("[WebSocket] Received message from %s to %s: %s (type: %s)",
			msg.Username, msg.Receiver, msg.Message, msg.Type)

		// Ensure message username matches the authenticated user
		if msg.Username != username {
			log.Printf("[WebSocket] Warning: Message username (%s) doesn't match authenticated user (%s)",
				msg.Username, username)
			msg.Username = username
		}

		sendMessageToRecipient(msg)
		notification.CreateNotificationMessage(msg.Receiver, msg.Username, "message", msg.Message)
		if err := saveMessageToDB(msg.Username, msg.Receiver, msg.Message, msg.Type); err != nil {
			log.Printf("[WebSocket] Error saving message to DB: %v", err)
			errorMsg := map[string]string{
				"error": fmt.Sprintf("Failed to save message: %v", err),
				"type":  "error",
			}
			conn.WriteJSON(errorMsg)
		}
	}

	clientsMutex.Lock()
	conns := clients[username]
	for i, c := range conns {
		if c == conn {
			clients[username] = append(conns[:i], conns[i+1:]...) // Remove this connection
			break
		}
	}
	if len(clients[username]) == 0 { // Only mark offline if no connections remain
		delete(onlineUsers, username)
	}
	clientsMutex.Unlock()

	log.Printf("[WebSocket] User %s disconnected (remaining connections: %d)", username, len(clients[username]))
	broadcastOnlineUsers()
}

func saveMessageToDB(sender string, receiver string, message string, typee string) error {
	log.Printf("[saveMessageToDB] Starting message save - sender: %s, receiver: %s, type: %s, message: %s",
		sender, receiver, typee, message)

	if sender == "" {
		return fmt.Errorf("sender username cannot be empty")
	}
	if receiver == "" {
		return fmt.Errorf("receiver username cannot be empty")
	}
	if message == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	senderID, err := session.GetUserIDFromUsername(sender)
	if err != nil {
		log.Printf("[saveMessageToDB] Failed to get sender ID for username %s: %v", sender, err)
		return fmt.Errorf("failed to get sender ID: %w", err)
	}
	log.Printf("[saveMessageToDB] Got sender ID: %s", senderID)

	receiverID, err := session.GetUserIDFromUsername(receiver)
	if err != nil {
		log.Printf("[saveMessageToDB] Failed to get receiver ID for username %s: %v", receiver, err)
		return fmt.Errorf("failed to get receiver ID: %w", err)
	}
	log.Printf("[saveMessageToDB] Got receiver ID: %s", receiverID)

	if senderID == receiverID {
		log.Printf("[saveMessageToDB] Error: sender and receiver are the same: %s", senderID)
		return fmt.Errorf("sender and receiver cannot be the same")
	}

	messageID, errr := uuid.NewV7()
	if errr != nil {
		log.Printf("[saveMessageToDB] Failed to generate message ID: %v", errr)
		return fmt.Errorf("failed to generate message ID: %w", errr)
	}
	log.Printf("[saveMessageToDB] Generated message ID: %s", messageID)

	if typee != "typing" {
		log.Printf("[saveMessageToDB] Preparing to insert message (type: %s)", typee)

		// First check if the users exist
		var senderExists, receiverExists bool
		err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", senderID).Scan(&senderExists)
		if err != nil {
			log.Printf("[saveMessageToDB] Error checking if sender exists: %v", err)
			return fmt.Errorf("error checking sender: %w", err)
		}
		err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", receiverID).Scan(&receiverExists)
		if err != nil {
			log.Printf("[saveMessageToDB] Error checking if receiver exists: %v", err)
			return fmt.Errorf("error checking receiver: %w", err)
		}

		if !senderExists {
			log.Printf("[saveMessageToDB] Error: sender ID %s does not exist in users table", senderID)
			return fmt.Errorf("sender does not exist")
		}
		if !receiverExists {
			log.Printf("[saveMessageToDB] Error: receiver ID %s does not exist in users table", receiverID)
			return fmt.Errorf("receiver does not exist")
		}

		// Prepare the insert statement
		pre, err := db.DB.Prepare("INSERT INTO messages (id, sender_id, receiver_id, content, creation_date) VALUES ($1, $2, $3, $4, $5)")
		if err != nil {
			log.Printf("[saveMessageToDB] Failed to prepare statement: %v", err)
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer pre.Close()

		// Execute the insert
		_, err = pre.Exec(messageID, senderID, receiverID, message, time.Now())
		if err != nil {
			log.Printf("[saveMessageToDB] Failed to execute statement: %v", err)
			return fmt.Errorf("failed to execute statement: %w", err)
		}
		log.Printf("[saveMessageToDB] Successfully inserted message with ID: %s", messageID)
	} else {
		log.Printf("[saveMessageToDB] Skipping message insertion for typing notification")
	}

	return nil
}

func GetMessages(w http.ResponseWriter, r *http.Request) {
	log.Println("[GetMessages] Starting message fetch request")
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		log.Println("[GetMessages] Invalid method:", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sessionn, err := r.Cookie("token")
	if err != nil {
		log.Println("[GetMessages] Token cookie error:", err)
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	token := sessionn.Value
	userid, ok := session.GetUserIDFromToken(token)
	if !ok || userid == "" {
		log.Println("[GetMessages] Invalid token or empty userid")
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	username, ok := session.GetUsernameFromUserID(userid)
	if !ok {
		log.Println("[GetMessages] Invalid username for userid:", userid)
		http.Error(w, "Unauthorized: Invalid username", http.StatusUnauthorized)
		return
	}

	sender := r.URL.Query().Get("sender")
	receiver := r.URL.Query().Get("receiver")
	log.Printf("[GetMessages] Request parameters - sender: %s, receiver: %s, current user: %s", sender, receiver, username)

	if sender == "" || receiver == "" {
		log.Println("[GetMessages] Missing sender or receiver")
		http.Error(w, "Sender and receiver are required", http.StatusBadRequest)
		return
	}

	if sender != username && receiver != username {
		log.Printf("[GetMessages] Unauthorized access attempt - sender: %s, receiver: %s, current user: %s", sender, receiver, username)
		http.Error(w, "You are not authorized to view these messages", http.StatusForbidden)
		return
	}

	senderID, ok1 := session.GetUserIDFromUsername(sender)
	if ok1 != nil {
		log.Printf("[GetMessages] Invalid sender username: %s, error: %v", sender, ok1)
		http.Error(w, "Invalid sender username", http.StatusBadRequest)
		return
	}

	receiverID, ok2 := session.GetUserIDFromUsername(receiver)
	if ok2 != nil {
		log.Printf("[GetMessages] Invalid receiver username: %s, error: %v", receiver, ok2)
		http.Error(w, "Invalid receiver username", http.StatusBadRequest)
		return
	}

	if senderID == receiverID {
		log.Println("[GetMessages] Same sender and receiver ID:", senderID)
		http.Error(w, "Sender and receiver cannot be the same", http.StatusBadRequest)
		return
	}

	log.Printf("[GetMessages] Executing query with senderID: %s, receiverID: %s", senderID, receiverID)
	rows, err := db.DB.Query(`
		SELECT sender_id, receiver_id, content, creation_date
		FROM messages
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $3 AND receiver_id = $4)
		ORDER BY creation_date ASC
	`, senderID, receiverID, receiverID, senderID)
	if err != nil {
		log.Printf("[GetMessages] Database query error: %v", err)
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
			log.Printf("[GetMessages] Error scanning row: %v", err)
			http.Error(w, "Error scanning message row", http.StatusInternalServerError)
			logger.LogError("Error scanning message row", err)
			return
		}

		senderUsername, ok := session.GetUsernameFromUserID(senderID)
		if !ok {
			log.Printf("[GetMessages] Error getting sender username for ID: %s", senderID)
			continue
		}

		receiverUsername, ok := session.GetUsernameFromUserID(receiverID)
		if !ok {
			log.Printf("[GetMessages] Error getting receiver username for ID: %s", receiverID)
			continue
		}

		messages = append(messages, Message{
			Username: senderUsername,
			Message:  content,
			Receiver: receiverUsername,
			Time:     creationDate,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("[GetMessages] Error iterating rows: %v", err)
		http.Error(w, "Error iterating over rows", http.StatusInternalServerError)
		logger.LogError("Error iterating over rows", err)
		return
	}

	log.Printf("[GetMessages] Successfully fetched %d messages", len(messages))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		log.Printf("[GetMessages] Error encoding response: %v", err)
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

	for _, conn := range recipientConns { // Send to all connections of the recipient
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
		// Get all users except the current client
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

// GetAllUsersExceptCurrent returns all users except the specified username
func GetAllUsersExceptCurrent(currentUsername string) ([]string, error) {
	rows, err := db.DB.Query("SELECT username FROM users WHERE username != $1", currentUsername)
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
