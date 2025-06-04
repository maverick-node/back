package messages

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"social-net/db"
	"social-net/notification"
	"social-net/session"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

var (
	groupUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Origin") == "https://frontend-social-net.vercel.app"
		},
	}

	// Map to store group connections
	groupConnections = make(map[string]map[string]*websocket.Conn)
	groupMutex       = &sync.Mutex{}
)

type GroupMessage struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	SenderID  string    `json:"sender_id"`
	Avatar    string    `json:"avatar"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type GroupMessageRequest struct {
	GroupID string `json:"group_id"`
	Content string `json:"content"`
}

func HandleGroupWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get current user
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "No authentication token", http.StatusUnauthorized)
		return
	}

	userID, ok := session.GetUserIDFromToken(token.Value)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := groupUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Get group ID from URL path
	groupID := strings.TrimPrefix(r.URL.Path, "/ws/group/")
	fmt.Println("groupID", groupID)
	if groupID == "" {
		errorMsg := map[string]string{"error": "Group ID is required"}
		conn.WriteJSON(errorMsg)
		return
	}

	// Verify user is a member of the group
	if !isGroupMember(userID, groupID) {
		errorMsg := map[string]string{"error": "Not a member of this group"}
		conn.WriteJSON(errorMsg)
		return
	}

	//group exist
	groupExist := isGroupExist(groupID)
	if !groupExist {
		errorMsg := map[string]string{"error": "Group does not exist"}
		conn.WriteJSON(errorMsg)
		return
	}

	// Add connection to group connections
	groupMutex.Lock()
	if groupConnections[groupID] == nil {
		groupConnections[groupID] = make(map[string]*websocket.Conn)
	}
	groupConnections[groupID][userID] = conn
	groupMutex.Unlock()

	// Remove connection when function returns
	defer func() {
		groupMutex.Lock()
		delete(groupConnections[groupID], userID)
		if len(groupConnections[groupID]) == 0 {
			delete(groupConnections, groupID)
		}
		groupMutex.Unlock()
	}()

	// Send recent messages
	sendRecentMessages(conn, groupID)

	// Handle incoming messages
	for {
		var msgReq GroupMessageRequest
		err := conn.ReadJSON(&msgReq)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		// Add length validation for group messages
		if len(msgReq.Content) < 1 {
			errorMsg := map[string]string{"error": "Message must be at least 1 character long"}
			conn.WriteJSON(errorMsg)
			continue
		}

		if len(msgReq.Content) > 500 {
			errorMsg := map[string]string{"error": "Message must not exceed 500 characters"}
			conn.WriteJSON(errorMsg)
			continue
		}

		// Create new message
		messageID, err := uuid.NewV4()
		if err != nil {
			log.Printf("Error generating UUID: %v", err)
			continue
		}

		msg := GroupMessage{
			ID:        messageID.String(),
			GroupID:   groupID,
			SenderID:  userID,
			Content:   msgReq.Content,
			CreatedAt: time.Now(),
		}

		// Save message to database
		if err := saveGroupMessage(msg); err != nil {
			log.Printf("Error saving message: %v", err)
			continue
		}
		members, err := getGroupMembers(groupID)
		if err != nil {
			log.Printf("Error getting group members: %v", err)
			continue
		}
		for _, member := range members {

			memberID, _ := session.GetUsernameFromUserID(member)
			senderid, _ := session.GetUsernameFromUserID(msg.SenderID)
			if memberID == senderid {
				continue
			}
			notification.CreateNotificationMessage(memberID, senderid, "group_message", msg.Content)
		}
		// Broadcast message to all group members
		broadcastGroupMessage(msg)
	}
}
func isGroupExist(groupID string) bool {
	var exists bool
	query := `
		SELECT EXISTS(SELECT 1 FROM groups WHERE id = ?)
	`
	err := db.DB.QueryRow(query, groupID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking group existence: %v", err)
		return false
	}
	return exists
}
func isGroupMember(userID, groupID string) bool {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM group_members 
			WHERE group_id = ? AND user_id = ? AND status = 'accepted'
		)
	`
	err := db.DB.QueryRow(query, groupID, userID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking group membership: %v", err)
		return false
	}
	return exists
}

func saveGroupMessage(msg GroupMessage) error {
	query := `
		INSERT INTO group_messages (id, group_id, sender_id, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.DB.Exec(query, msg.ID, msg.GroupID, msg.SenderID, msg.Content, msg.CreatedAt)
	return err
}

func sendRecentMessages(conn *websocket.Conn, groupID string) {
	query := `
		SELECT m.id, m.group_id, m.sender_id, m.content, m.created_at, u.username, u.avatar
		FROM group_messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.group_id = ?
		ORDER BY m.created_at DESC
		LIMIT 50
	`
	rows, err := db.DB.Query(query, groupID)
	if err != nil {
		log.Printf("Error fetching recent messages: %v", err)
		return
	}
	defer rows.Close()

	var messages []map[string]interface{}
	for rows.Next() {
		var msg GroupMessage
		var username string
		var avatar sql.NullString
		if err := rows.Scan(&msg.ID, &msg.GroupID, &msg.SenderID, &msg.Content, &msg.CreatedAt, &username, &avatar); err != nil {
			log.Printf("Error scanning message: %v", err)
			continue
		}

		// Set avatar URL or default
		var avatarURL string
		if avatar.Valid && avatar.String != "" {
			avatarURL = avatar.String
		} else {
			avatarURL = ""
		}

		messageMap := map[string]interface{}{
			"id":         msg.ID,
			"group_id":   msg.GroupID,
			"sender_id":  msg.SenderID,
			"content":    msg.Content,
			"created_at": msg.CreatedAt,
			"username":   username,
			"avatar":     avatarURL,
		}
		messages = append(messages, messageMap)
	}

	if err := conn.WriteJSON(messages); err != nil {
		log.Printf("Error sending recent messages: %v", err)
	}
}

func broadcastGroupMessage(msg GroupMessage) {
	// Get sender's username and avatar
	var username string
	var avatar sql.NullString
	err := db.DB.QueryRow("SELECT username, avatar FROM users WHERE id = ?", msg.SenderID).Scan(&username, &avatar)
	if err != nil {
		log.Printf("Error getting username and avatar: %v", err)
		return
	}

	// Set avatar URL or default
	var avatarURL string
	if avatar.Valid && avatar.String != "" {
		avatarURL = avatar.String
	} else {
		avatarURL = ""
	}

	// Prepare message for broadcast
	messageMap := map[string]interface{}{
		"id":         msg.ID,
		"group_id":   msg.GroupID,
		"sender_id":  msg.SenderID,
		"avatar":     avatarURL,
		"content":    msg.Content,
		"created_at": msg.CreatedAt,
		"username":   username,
	}

	// Broadcast to all connections in the group
	groupMutex.Lock()
	defer groupMutex.Unlock()

	if connections, ok := groupConnections[msg.GroupID]; ok {
		for _, conn := range connections {
			if err := conn.WriteJSON(messageMap); err != nil {
				log.Printf("Error broadcasting message: %v", err)
			}
		}
	}
}

func getGroupMembers(groupID string) ([]string, error) {
	query := `
		SELECT user_id FROM group_members WHERE group_id = ?
	`
	rows, err := db.DB.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var member string
		if err := rows.Scan(&member); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, nil
}
