package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"social-net/db"
	"social-net/session"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

var dbMutex sync.Mutex

var (
	notificationClients  = make(map[string][]*websocket.Conn)
	notificationMutex    sync.Mutex
	notificationUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

const (
	TypeMessage       = "message"
	TypeFollowRequest = "follow_request"
	TypeGroupInvite   = "group_invite"
	TypeGroupRequest  = "group_request"
	TypeEventCreated  = "event_created"
	TypeGroupMessage  = "group_message"
)

type NotificationWebSocketMessage struct {
	Type         string       `json:"type"`
	Notification Notification `json:"notification"`
}

func HandleNotificationWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := notificationUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading notification WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	sessionCookie, err := r.Cookie("token")
	if err != nil {
		log.Printf("Error getting token for notification WebSocket: %v", err)
		return
	}

	token := sessionCookie.Value
	userID, ok := session.GetUserIDFromToken(token)
	if !ok || userID == "" {
		log.Printf("Invalid token for notification WebSocket")
		return
	}

	username, ok := session.GetUsernameFromUserID(userID)
	if !ok {
		log.Printf("Invalid username for notification WebSocket")
		return
	}

	notificationMutex.Lock()
	notificationClients[username] = append(notificationClients[username], conn)
	notificationMutex.Unlock()

	log.Printf("User %s connected to notification WebSocket", username)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Notification WebSocket connection closed for user %s: %v", username, err)
			break
		}
	}

	notificationMutex.Lock()
	conns := notificationClients[username]
	for i, c := range conns {
		if c == conn {
			notificationClients[username] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(notificationClients[username]) == 0 {
		delete(notificationClients, username)
	}
	notificationMutex.Unlock()

	log.Printf("User %s disconnected from notification WebSocket", username)
}

func BroadcastNotificationToUser(username string, notification Notification) {
	notificationMutex.Lock()
	defer notificationMutex.Unlock()

	connections, exists := notificationClients[username]
	if !exists || len(connections) == 0 {
		log.Printf("No notification WebSocket connections for user %s", username)
		return
	}

	message := NotificationWebSocketMessage{
		Type:         "new_notification",
		Notification: notification,
	}

	for i, conn := range connections {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Failed to send notification to user %s connection %d: %v", username, i, err)

			notificationClients[username] = append(connections[:i], connections[i+1:]...)
		}
	}
}

func CreateNotificationMessage(userUS string, senderUS string, notifType string, content string) error {
	userID, _ := session.GetUserIDFromUsername(userUS)
	senderID, _ := session.GetUserIDFromUsername(senderUS)
	fmt.Println("XXXXXXX2")

	formattedContent := content

	notificationID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("failed to generate notification ID: %w", err)
	}
	fmt.Println("user", userID, "sender", senderID)

	dbMutex.Lock()
	defer dbMutex.Unlock()

	query := `
	INSERT INTO notifications (id, user_id, sender_id, type, content, is_read, created_at)
	VALUES (?, ?, ?, ?, ?, 0, ?)
	`

	createdAt := time.Now()
	_, err = db.DB.Exec(query, notificationID.String(), userID, senderID, notifType, formattedContent, createdAt)
	if err != nil {
		fmt.Println("Error generating notification ID:", err)
		return fmt.Errorf("failed to insert notification: %w", err)
	}
	fmt.Println("XXXXXXX3")

	notification := Notification{
		ID:             notificationID.String(),
		UserID:         userID,
		SenderID:       senderID,
		Type:           notifType,
		Content:        formattedContent,
		IsRead:         false,
		CreatedAt:      createdAt,
		SenderUsername: senderUS,
	}

	go BroadcastNotificationToUser(userUS, notification)

	return nil
}

type Notification struct {
	ID                string         `json:"id"`
	UserID            string         `json:"user_id"`
	SenderID          string         `json:"sender_id"`
	Type              string         `json:"type"`
	Content           string         `json:"content"`
	IsRead            bool           `json:"is_read"`
	CreatedAt         time.Time      `json:"created_at"`
	SenderUsername    string         `json:"sender_username"`
	RelatedEntityID   sql.NullString `json:"related_entity_id"`
	RelatedEntityType sql.NullString `json:"related_entity_type"`
}

func GetNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://white-pebble-0a50c5603.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	token, _ := r.Cookie("token")
	tok := token.Value
	userID, ok := session.GetUserIDFromToken(tok)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	query := `
		SELECT 
			n.id,
			n.user_id,
			n.sender_id,
			n.type,
			n.content,
			n.is_read,
			n.created_at,
			u.username as sender_username
		FROM notifications n
		LEFT JOIN users u ON n.sender_id = u.id
		WHERE n.user_id = $1
		ORDER BY n.created_at DESC
	`

	rows, err := db.DB.Query(query, userID)
	if err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type NotificationResponse struct {
		ID             string    `json:"id"`
		UserID         string    `json:"user_id"`
		SenderID       string    `json:"sender_id"`
		SenderUsername string    `json:"sender_username"`
		Type           string    `json:"type"`
		Content        string    `json:"content"`
		IsRead         bool      `json:"is_read"`
		CreatedAt      time.Time `json:"created_at"`
	}

	notifications := []NotificationResponse{}
	for rows.Next() {
		var n NotificationResponse
		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.SenderID,
			&n.Type,
			&n.Content,
			&n.IsRead,
			&n.CreatedAt,
			&n.SenderUsername,
		)
		if err != nil {
			continue
		}
		notifications = append(notifications, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://white-pebble-0a50c5603.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var requestBody struct {
		NotificationID string `json:"notificationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	fmt.Println("requestBody", requestBody)

	token, _ := r.Cookie("token")
	userID, _ := session.GetUserIDFromToken(token.Value)

	query := `
		UPDATE notifications 
		SET is_read = 1 
		WHERE id = ? AND user_id = ?
	`
	_, err := db.DB.Exec(query, requestBody.NotificationID, userID)
	if err != nil {
		http.Error(w, "Failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteNotification(users string, sender string, notificationtype string) {
	query := `
		DELETE FROM notifications 
		WHERE user_id = ? AND sender_id = ? AND type = ?
	`
	_, err := db.DB.Exec(query, users, sender, notificationtype)
	if err != nil {
		fmt.Println("Error deleting notification:", err)
		return
	}
}
