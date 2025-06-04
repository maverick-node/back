package notification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"social-net/db"
	"social-net/session"
	"sync"
	"time"

	"github.com/gofrs/uuid"
)

var dbMutex sync.Mutex

// NotificationType constants
const (
	TypeMessage       = "message"
	TypeFollowRequest = "follow_request"
	TypeGroupInvite   = "group_invite"
	TypeGroupRequest  = "group_request"
	TypeEventCreated  = "event_created"
	TypeGroupMessage  = "group_message"
)

// CreateNotificationMessage creates a new notification with type and content
func CreateNotificationMessage(userUS string, senderUS string, notifType string, content string) error {
	userID, _ := session.GetUserIDFromUsername(userUS)
	senderID, _ := session.GetUserIDFromUsername(senderUS)
	fmt.Println("XXXXXXX2")
	// Format content based on notification type
	formattedContent := content

	// Generate UUID for the notification
	notificationID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("failed to generate notification ID: %w", err)
	}
	fmt.Println("user", userID, "sender", senderID)

	// Lock the mutex before database operations
	dbMutex.Lock()
	defer dbMutex.Unlock()

	query := `
	INSERT INTO notifications (id, user_id, sender_id, type, content, is_read, created_at)
	VALUES (?, ?, ?, ?, ?, 0, ?)
	`

	_, err = db.DB.Exec(query, notificationID.String(), userID, senderID, notifType, formattedContent, time.Now())
	if err != nil {
		fmt.Println("Error generating notification ID:", err)
		return fmt.Errorf("failed to insert notification: %w", err)
	}
	fmt.Println("XXXXXXX3")
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
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-net.vercel.app")
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
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-net.vercel.app")
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

// DeleteNotification deletes a notification
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
