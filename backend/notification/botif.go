package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"social-net/db"
	"social-net/session"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

var dbMutex sync.Mutex

// Add WebSocket connection management for notifications
var (
	notificationClients  = make(map[string][]*websocket.Conn)
	notificationMutex    sync.Mutex
	notificationUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// NotificationType constants
const (
	TypeMessage       = "message"
	TypeFollowRequest = "follow_request"
	TypeGroupInvite   = "group_invite"
	TypeGroupRequest  = "group_request"
	TypeEventCreated  = "event_created"
	TypeGroupMessage  = "group_message"
)

// WebSocket message structure for notifications
type NotificationWebSocketMessage struct {
	Type         string       `json:"type"`
	Notification Notification `json:"notification"`
}

// HandleNotificationWebSocket handles WebSocket connections for real-time notifications
func HandleNotificationWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := notificationUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading notification WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	// Get user from session
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

	// Register the WebSocket connection
	notificationMutex.Lock()
	notificationClients[username] = append(notificationClients[username], conn)
	notificationMutex.Unlock()

	log.Printf("User %s connected to notification WebSocket", username)

	// Keep connection alive and handle disconnection
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Notification WebSocket connection closed for user %s: %v", username, err)
			break
		}
	}

	// Remove connection on disconnect
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

// BroadcastNotificationToUser sends a real-time notification to a specific user
func BroadcastNotificationToUser(username string, notification Notification) {
	notificationMutex.Lock()
	defer notificationMutex.Unlock()

	connections, exists := notificationClients[username]
	if !exists || len(connections) == 0 {
		// This is not an error condition - the user is just not currently connected
		// They will receive the notification when they reconnect and fetch notifications
		log.Printf("User %s is not currently connected to WebSocket. Notification will be available when they reconnect.", username)
		return
	}

	message := NotificationWebSocketMessage{
		Type:         "new_notification",
		Notification: notification,
	}

	// Send to all connections for this user
	for i, conn := range connections {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Failed to send notification to user %s connection %d: %v", username, i, err)
			// Remove failed connection
			notificationClients[username] = append(connections[:i], connections[i+1:]...)
		}
	}
}

// CreateNotificationMessage creates a new notification with type and content
func CreateNotificationMessage(userUS string, senderUS string, notifType string, content string) error {
	if userUS == "" {
		return fmt.Errorf("recipient username cannot be empty")
	}
	if senderUS == "" {
		return fmt.Errorf("sender username cannot be empty")
	}

	userID, err := session.GetUserIDFromUsername(userUS)
	if err != nil {
		return fmt.Errorf("failed to get recipient user ID: %w", err)
	}

	senderID, err := session.GetUserIDFromUsername(senderUS)
	if err != nil {
		return fmt.Errorf("failed to get sender user ID: %w", err)
	}

	// Format content based on notification type
	formattedContent := content

	// Generate UUID for the notification
	notificationID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("failed to generate notification ID: %w", err)
	}

	// Lock the mutex before database operations
	dbMutex.Lock()
	defer dbMutex.Unlock()

	query := `
	INSERT INTO notifications (id, user_id, sender_id, type, content, is_read, created_at)
	VALUES ($1, $2, $3, $4, $5, false, $6)
	`

	createdAt := time.Now()
	_, err = db.DB.Exec(query, notificationID.String(), userID, senderID, notifType, formattedContent, createdAt)
	if err != nil {
		return fmt.Errorf("failed to insert notification: %w", err)
	}

	// Create notification object for real-time broadcast
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

	// Broadcast real-time notification
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

// GetNotifications retrieves notifications for a user
func GetNotifications(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get user ID from token
	token, _ := r.Cookie("token")
	tok := token.Value
	userID, ok := session.GetUserIDFromToken(tok)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Query to get notifications with sender username
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

// MarkNotificationAsRead marks a notification as read
func MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse request body
	var requestBody struct {
		NotificationID string `json:"notificationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	fmt.Println("requestBody", requestBody)

	// Get user ID from token
	token, _ := r.Cookie("token")
	userID, _ := session.GetUserIDFromToken(token.Value)

	// Update notification
	query := `
		UPDATE notifications 
		SET is_read = true 
		WHERE id = $1 AND user_id = $2
	`
	_, err := db.DB.Exec(query, requestBody.NotificationID, userID)
	if err != nil {
		http.Error(w, "Failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteNotification deletes a notification
func DeleteNotification(users string, sender string, notificationtype string) {
	query := `
		DELETE FROM notifications 
		WHERE user_id = $1 AND sender_id = $2 AND type = $3
	`
	_, err := db.DB.Exec(query, users, sender, notificationtype)
	if err != nil {
		fmt.Println("Error deleting notification:", err)
		return
	}
}
