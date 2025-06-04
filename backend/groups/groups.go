package groups

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"social-net/db"
	logger "social-net/log"
	"social-net/notification"

	"social-net/session"

	"github.com/gofrs/uuid"
)

type Group struct {
	ID          string `json:"id"`
	CreatorID   string `json:"creator_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Members     []int  `json:"members"`
}

type GroupMember struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
	Status  string `json:"status"` // "pending", "accepted", "declined"
}

type GroupPost struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	GroupID      string    `json:"group_id"`
	UserID       string    `json:"user_id"`
	Author       string    `json:"author"`
	Content      string    `json:"content"`
	Image        string    `json:"image"`
	CreationDate time.Time `json:"creation_date"`
	Avatar       string    `json:"avatar"`
}

func CreateGroup(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST method
	if r.Method != http.MethodPost {
		fmt.Println("Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var group Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		logger.LogError("Invalid request body", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if group.Title == "" || group.Description == "" {
		logger.LogError("Title and description are required", nil)
		http.Error(w, "Title and description are required", http.StatusBadRequest)
		return
	}

	if len(group.Title) > 50 || len(group.Description) > 500 {
		logger.LogError("Title or description too long", nil)
		http.Error(w, "Title must be less than 50 characters and description less than 500 characters", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO groups (id,creator_id, title, description)
		VALUES ($1, $2, $3,$4)
	`
	token, _ := r.Cookie("token")
	tok := token.Value
	userid, ok := session.GetUserIDFromToken(tok)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	groupID, err := uuid.NewV7()
	if err != nil {
		fmt.Println("Failed to generate group ID", err)
		http.Error(w, "Failed to generate group ID", http.StatusInternalServerError)
		return
	}
	group.ID = groupID.String()
	group.CreatorID = userid
	_, err = db.DB.Exec(query, groupID, userid, group.Title, group.Description)
	if err != nil {
		logger.LogError("Failed to create group", err)
		http.Error(w, "Failed to create group", http.StatusInternalServerError)
		return
	}

	// Add creator as first member with accepted status
	memberQuery := "INSERT INTO group_members (group_id, user_id, is_admin, status) VALUES ($1, $2, '1', 'accepted')"
	_, err = db.DB.Exec(memberQuery, groupID.String(), userid)
	if err != nil {
		logger.LogError("Faild to add creator to group", err)
		http.Error(w, "Failed to add creator to group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

func GetGroup(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow GET method
	if r.Method != http.MethodGet {
		logger.LogError("meethod not allowed", nil)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get group ID from query parameters
	groupID := r.URL.Query().Get("id")
	if groupID == "" {
		logger.LogError("Group ID is required", nil)
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	query := "SELECT id, creator_id, title, description FROM groups WHERE id = $1"
	group := &Group{}
	err := db.DB.QueryRow(query, groupID).Scan(&group.ID, &group.CreatorID, &group.Title, &group.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.LogError("Group not found", err)

			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			logger.LogError("Failed to fetch group", err)
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

func AddMemberToGroup(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get group ID and user ID from request body
	var request struct {
		GroupID string `json:"group_id"`
		UserID  string `json:"user_id"`
		Status  string `json:"status"`
	}

	token, _ := r.Cookie("token")
	tok := token.Value
	userid, ok := session.GetUserIDFromToken(tok)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.LogError("Invalid request", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	fmt.Println("Adding member to group:", request.GroupID, request.UserID, request.Status)
	if request.GroupID == "" || request.UserID == "" || request.Status == "" {
		http.Error(w, "Group ID, User ID, and Status are required", http.StatusBadRequest)
		return
	}

	if request.Status != "pending" && request.Status != "accepted" && request.Status != "invited" {
		http.Error(w, "Invalid status. Must be 'pending', 'invited', or 'accepted'", http.StatusBadRequest)
		return
	}
	fmt.Println("Adding member to group:", request.GroupID, request.UserID, request.Status)
	// check if invited user exist
	var userExists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", request.UserID).Scan(&userExists)
	if err != nil {
		fmt.Println("Failed to check if user exists", err)
		http.Error(w, "Failed to check if user exists", http.StatusInternalServerError)
		return
	}
	if !userExists {
		http.Error(w, "User does not exist", http.StatusNotFound)
		return
	}
	// Check if user is already a member
	var exists bool
	var currentStatus string
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2), COALESCE((SELECT status FROM group_members WHERE group_id = $1 AND user_id = $2), '') as status",
		request.GroupID, request.UserID).Scan(&exists, &currentStatus)
	if err != nil {
		fmt.Println("Failed to check existing membership", err)
		http.Error(w, "Failed to check existing membership", http.StatusInternalServerError)
		return
	}

	if exists {
		// If user is already invited or has a pending request, don't allow another invitation
		if currentStatus == "invited" || currentStatus == "pending" {
			http.Error(w, fmt.Sprintf("User already has a %s request", currentStatus), http.StatusConflict)
			return
		}
		// If user is already an accepted member, don't allow invitation
		if currentStatus == "accepted" {
			http.Error(w, "User is already a member of the group", http.StatusConflict)
			return
		}
	}

	// Insert the new member with the specified status
	query := "INSERT INTO group_members (group_id, user_id, is_admin, status) VALUES ($1, $2, '0', $3)"
	_, err = db.DB.Exec(query, request.GroupID, request.UserID, request.Status)
	if err != nil {
		logger.LogError("Failed to add member to group", err)
		http.Error(w, "Failed to add member to group", http.StatusInternalServerError)
		return
	}

	query1 := "select title from groups where id = $1"
	var groupName string
	err = db.DB.QueryRow(query1, request.GroupID).Scan(&groupName)
	if err != nil {
		fmt.Println("Failed to fetch group name", err)
		http.Error(w, "Failed to fetch group name", http.StatusInternalServerError)
		return
	}

	var ownerFirstName, ownerLastName string
	err = db.DB.QueryRow("SELECT first_name, last_name FROM users WHERE id = $1", userid).Scan(&ownerFirstName, &ownerLastName)
	if err != nil {
		fmt.Println("Failed to fetch owner's name", err)
		http.Error(w, "Failed to fetch owner's name", http.StatusInternalServerError)
		return
	}
	ownerFullName := ownerFirstName + " " + ownerLastName

	ownerUsername, _ := session.GetUsernameFromUserID(userid)
	userUsername, _ := session.GetUsernameFromUserID(request.UserID)
	notification.CreateNotificationMessage(userUsername, ownerUsername, "Group", ownerFullName+" has invited you to join the group "+groupName)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Member added successfully"))
}

func AcceptGroupMember(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST method
	if r.Method != http.MethodPost {
		logger.LogError("Method not allowed", nil)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get group ID and user ID from request body
	var request struct {
		GroupID string `json:"group_id"`
		UserID  string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.LogError("Invalid request body", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update member status to accepted
	query := "UPDATE group_members SET status = 'accepted' WHERE group_id = $1 AND user_id = $2 AND status = 'pending'"
	_, err := db.DB.Exec(query, request.GroupID, request.UserID)
	if err != nil {
		logger.LogError("Faild to accept group member", err)
		http.Error(w, "Failed to accept group member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Member accepted successfully"))
}

func DeclineGroupMember(db *sql.DB, groupID int, userID int) error {
	query := "DELETE FROM group_members WHERE group_id = $1 AND user_id = $2 AND status = 'pending'"
	_, err := db.Exec(query, groupID, userID)

	return err
}

func GetPendingMembers(db *sql.DB, groupID int) ([]int, error) {
	query := "SELECT user_id FROM group_members WHERE group_id = $1 AND status = 'pending'"
	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []int{}
	for rows.Next() {
		var userID int
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}
		members = append(members, userID)
	}
	return members, nil
}

func GetAcceptedMembers(db *sql.DB, groupID int) ([]int, error) {
	query := "SELECT user_id FROM group_members WHERE group_id = $1 AND status = 'accepted'"
	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []int{}
	for rows.Next() {
		var userID int
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}
		members = append(members, userID)
	}
	return members, nil
}

func RemoveMemberFromGroup(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get group ID and user ID from request body
	var request struct {
		GroupID string `json:"group_id"`
		UserID  string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Request body", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, _ := r.Cookie("token")
	tok := token.Value
	userid, ok := session.GetUserIDFromToken(tok)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// First check if user is the group owner
	var isOwner bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1 AND creator_id = $2)", request.GroupID, userid).Scan(&isOwner)
	if err != nil {
		http.Error(w, "Failed to check group ownership", http.StatusInternalServerError)
		return
	}

	if isOwner && request.UserID == userid {
		http.Error(w, "Group owner cannot leave their own group", http.StatusForbidden)
		return
	}
	fmt.Println("Request body", request)
	fmt.Println("User id", userid)
	fmt.Println("Group id", request.GroupID)
	fmt.Println("User id", request.UserID)
	// Delete the member from the group
	result, err := db.DB.Exec("DELETE FROM group_members WHERE group_id = $1 AND user_id = $2", request.GroupID, request.UserID)
	if err != nil {
		http.Error(w, "Failed to remove member from group", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to check if member was removed", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Member not found in group or is Owner", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Member removed successfully"))
}

func GetGroupMembers(db *sql.DB, groupID int) ([]int, error) {
	query := "SELECT user_id FROM group_members WHERE group_id = $1"
	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []int{}
	for rows.Next() {
		var userID int
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}
		members = append(members, userID)
	}
	return members, nil
}

func GetGroups(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user's ID from token
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	tok := token.Value
	currentUserID, ok := session.GetUserIDFromToken(tok)
	if !ok || currentUserID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Modified query to include membership status for the current user
	query := `
		SELECT 
			g.id, 
			g.creator_id, 
			g.title, 
			g.description,
			COALESCE(gm.status, 'not_member') as member_status,
			COALESCE(gm.is_admin, '0') as is_admin
		FROM groups g
		LEFT JOIN group_members gm ON g.id = gm.group_id AND gm.user_id = $1
		ORDER BY g.id DESC`

	rows, err := db.DB.Query(query, currentUserID)
	if err != nil {
		http.Error(w, "Failed to fetch groups", http.StatusInternalServerError)
		logger.LogError("Failed to fetch groups", err)
		return
	}
	defer rows.Close()

	type GroupWithStatus struct {
		Group
		MemberStatus string `json:"member_status"`
		IsOwner      bool   `json:"is_owner"`
	}

	groups := []GroupWithStatus{}
	for rows.Next() {
		var group GroupWithStatus
		var ownerID string
		err := rows.Scan(&group.ID, &ownerID, &group.Title, &group.Description, &group.MemberStatus, &group.IsOwner)
		if err != nil {
			http.Error(w, "Failed to scan group", http.StatusInternalServerError)
			return
		}
		group.CreatorID, _ = session.GetUsernameFromUserID(ownerID)
		groups = append(groups, group)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

func MyGroups(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	token, _ := r.Cookie("token")
	tok := token.Value
	userid, ok := session.GetUserIDFromToken(tok)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Updated query to get all groups where user is a member with accepted status
	query := `
		SELECT g.id, g.creator_id, g.title, g.description, 
			   CASE WHEN g.creator_id = $1 THEN true ELSE false END as is_owner
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1 AND gm.status = 'accepted'
	`

	rows, err := db.DB.Query(query, userid)
	if err != nil {
		fmt.Println("Failed to fetch groups", err)
		http.Error(w, "Failed to fetch groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type GroupWithOwnership struct {
		Group
		IsOwner bool `json:"is_owner"`
	}

	groups := []GroupWithOwnership{}
	for rows.Next() {
		var group GroupWithOwnership
		err := rows.Scan(&group.ID, &group.CreatorID, &group.Title, &group.Description, &group.IsOwner)
		if err != nil {
			http.Error(w, "Failed to scan group", http.StatusInternalServerError)
			return
		}
		groups = append(groups, group)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

func ShowRequests(w http.ResponseWriter, r *http.Request) {
	token, _ := r.Cookie("token")
	tok := token.Value
	userid, ok := session.GetUserIDFromToken(tok)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	query := "SELECT id, creator_id, title, description FROM groups WHERE creator_id = $1"
	rows, err := db.DB.Query(query, userid)
	if err != nil {
		http.Error(w, "Failed to fetch groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	groups := []Group{}
	for rows.Next() {
		var group Group
		err := rows.Scan(&group.ID, &group.CreatorID, &group.Title, &group.Description)
		if err != nil {
			http.Error(w, "Failed to scan group", http.StatusInternalServerError)
			return
		}
		groups = append(groups, group)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

// GetPendingInvitations handles fetching pending group invitations for a user
func GetPendingInvitations(w http.ResponseWriter, r *http.Request) {
	fmt.Println("=== GetPendingInvitations called ===")

	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	token, _ := r.Cookie("token")
	tok := token.Value
	userID, ok := session.GetUserIDFromToken(tok)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	if userID == "" {
		fmt.Println("Error: Unauthorized - Invalid user ID")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get group_id from query parameters
	groupID := r.URL.Query().Get("group_id")
	fmt.Println("Requested group ID:", groupID)

	if groupID == "" {
		fmt.Println("Error: Group ID is required")
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Query to get users with pending status in the specified group
	query := `
		SELECT u.id, u.username, u.email, u.first_name, u.last_name
		FROM users u
		JOIN group_members gm ON u.id = gm.user_id
		WHERE gm.group_id = $1 AND gm.status = 'pending'
	`

	rows, err := db.DB.Query(query, groupID)
	if err != nil {
		fmt.Println("Error executing query:", err)
		http.Error(w, "Error fetching pending users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type PendingUser struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		FullName  string `json:"full_name"`
	}

	var pendingUsers []PendingUser
	for rows.Next() {
		var user PendingUser
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			http.Error(w, "Error scanning user", http.StatusInternalServerError)
			return
		}
		user.FullName = user.FirstName + " " + user.LastName
		fmt.Println("Found pending user:", user)
		pendingUsers = append(pendingUsers, user)
	}

	fmt.Println("Total pending users found:", len(pendingUsers))
	fmt.Println("Pending users for group", groupID, ":", pendingUsers)
	fmt.Println("=== GetPendingInvitations completed ===")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pendingUsers)
}

// HandleInvitation handles accepting or declining a group invitation
func HandleInvitation(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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

	// Parse request body
	var request struct {
		GroupID string `json:"group_id"`
		Action  string `json:"action"` // "accept" or "decline"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.LogError("Invalid request body", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var query string
	if request.Action == "accept" {
		query = `UPDATE group_members SET status = 'accepted' WHERE group_id = $1 AND user_id = $2 AND (status = 'pending' OR status = 'invited')`
	} else if request.Action == "decline" {
		query = `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2 AND (status = 'pending' OR status = 'invited')`
	} else {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	_, err := db.DB.Exec(query, request.GroupID, userID)
	if err != nil {
		logger.LogError("Error handling invitation", err)
		http.Error(w, "Error handling invitation", http.StatusInternalServerError)
		return
	}

	// Even if no rows were affected, we'll consider it a success
	// This handles the case where the status was already changed

	// Create notification for the group owner if accepting
	if request.Action == "accept" {
		// Get group owner ID
		var ownerID string
		err := db.DB.QueryRow("SELECT creator_id FROM groups WHERE id = $1", request.GroupID).Scan(&ownerID)
		if err == nil {
			ownerUsername, _ := session.GetUsernameFromUserID(ownerID)
			userUsername, _ := session.GetUsernameFromUserID(userID)
			notification.CreateNotificationMessage(
				ownerUsername,
				userUsername,
				notification.TypeGroupRequest,
				"accepted the invitation to join your group",
			)
		}
	}

	// Always return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Invitation %sed successfully", request.Action),
	})
}

// GetGroupInvitations handles fetching pending invitations for a specific group
func GetGroupInvitations(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get group ID from query parameters
	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Query to get pending invitations with user details
	query := `
		SELECT u.id, u.username, u.email, gm.created_at
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1 AND gm.status = 'pending'
	`

	rows, err := db.DB.Query(query, groupID)
	if err != nil {
		http.Error(w, "Error fetching group invitations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type PendingUser struct {
		ID        string    `json:"id"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
	}

	var pendingUsers []PendingUser
	for rows.Next() {
		var user PendingUser
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
		if err != nil {
			http.Error(w, "Error scanning user", http.StatusInternalServerError)
			return
		}
		pendingUsers = append(pendingUsers, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pendingUsers)
}

// HandleGroupInvitation handles accepting or declining a group invitation
func HandleGroupInvitation(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Parse request body
	var request struct {
		GroupID string `json:"group_id"`
		UserID  string `json:"user_id"`
		Action  string `json:"action"` // "accept" or "decline"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update group member status based on action
	var status string
	if request.Action == "accept" {
		status = "accepted"
	} else {
		status = "declined"
	}

	query := `
		UPDATE group_members
		SET status = $1
		WHERE group_id = $2 AND user_id = $3 AND status = 'pending'
	`

	result, err := db.DB.Exec(query, status, request.GroupID, request.UserID)
	if err != nil {
		http.Error(w, "Error updating invitation status", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error checking update result", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "No pending invitation found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Invitation %sed successfully", request.Action),
	})
}

// IsGroupMember checks if a user is a member of a specific group
func IsGroupMember(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get user ID from session
	token, _ := r.Cookie("token")
	tok := token.Value
	userID, ok := session.GetUserIDFromToken(tok)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Get group ID from query parameters
	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Query to check if user is a member of the group
	query := `
		SELECT status 
		FROM group_members 
		WHERE group_id = $1 AND user_id = $2
	`

	var status string
	err := db.DB.QueryRow(query, groupID, userID).Scan(&status)

	response := struct {
		IsMember bool   `json:"is_member"`
		Status   string `json:"status,omitempty"`
	}{}

	if err == sql.ErrNoRows {
		// User is not a member
		response.IsMember = false
	} else if err != nil {
		http.Error(w, "Error checking group membership", http.StatusInternalServerError)
		return
	} else {
		// User is a member
		response.IsMember = true
		response.Status = status
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckGroupMembershipStatus checks if a user is accepted in a group
func CheckGroupMembershipStatus(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get group_id and user_id from query parameters
	groupID := r.URL.Query().Get("group_id")
	token, _ := r.Cookie("token")
	tokenValue := token.Value
	userID, ok := session.GetUserIDFromToken(tokenValue)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	if groupID == "" || userID == "" {
		http.Error(w, "Missing group_id or user_id", http.StatusBadRequest)
		return
	}

	// Query the database to check membership status
	var status string
	err := db.DB.QueryRow(`
		SELECT status 
		FROM group_members 
		WHERE group_id = $1 AND user_id = $2
	`, groupID, userID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			// User is not a member of the group
			json.NewEncoder(w).Encode(map[string]interface{}{
				"is_member": false,
				"status":    "not_member",
			})
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Return the membership status
	json.NewEncoder(w).Encode(map[string]interface{}{
		"is_member": true,
		"status":    status,
	})
}

// Group Posts handling

func AddGroupPost(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}
	fmt.Println("Group ID:", groupID)
	// Get user ID from session
	token, _ := r.Cookie("token")
	tok := token.Value
	userID, ok := session.GetUserIDFromToken(tok)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Get post details from request body
	var post GroupPost
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		log.Println("[AddGroupPost] Error decoding request body:", err)
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	// Add length validation for group post title and content
	if len(post.Title) < 3 {
		log.Println("Group post title too short")
		http.Error(w, "Title must be at least 3 characters long", http.StatusBadRequest)
		return
	}

	if len(post.Title) > 100 {
		log.Println("Group post title too long")
		http.Error(w, "Title must not exceed 100 characters", http.StatusBadRequest)
		return
	}

	if len(post.Content) < 10 {
		log.Println("Group post content too short")
		http.Error(w, "Content must be at least 10 characters long", http.StatusBadRequest)
		return
	}

	if len(post.Content) > 1000 {
		log.Println("Group post content too long")
		http.Error(w, "Content must not exceed 1000 characters", http.StatusBadRequest)
		return
	}

	post_id, err := uuid.NewV7()
	if err != nil {
		log.Println("Failed to generate UUID:", err)
		http.Error(w, "Unknown Internal Error, Try again", http.StatusInternalServerError)
		return
	}
	var imagePath string
	if post.Image != "" {
		imagePath, err = saveBase64Image(post.Image, post_id.String())
		if err != nil {
			log.Println("[AddGroupPost] Error saving image:", err)
			http.Error(w, "Failed to save image", http.StatusInternalServerError)
			return
		}
	}
	// Insert post into database
	_, err = db.DB.Exec(`
		INSERT INTO group_posts (id, group_id, user_id, title, content, creation_date,image)
		VALUES ($1, $2, $3, $4, $5, $6,$7)
	`, post_id.String(), groupID, userID, post.Title, post.Content, time.Now().UTC(), imagePath)
	if err != nil {
		log.Println("[AddGroupPost] Error inserting post into database:", err)
		http.Error(w, "Failed to insert post into database", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusCreated)
}

func saveBase64Image(base64Data, postID string) (string, error) {
	// Check if the string contains base64 data
	if !strings.Contains(base64Data, ",") {
		return "", fmt.Errorf("invalid base64 image format")
	}

	// Split the base64 string to get the data part
	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid base64 image format")
	}

	// Get the file extension from the header
	header := parts[0]
	var ext string
	if strings.Contains(header, "jpeg") || strings.Contains(header, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(header, "png") {
		ext = ".png"
	} else if strings.Contains(header, "gif") {
		ext = ".gif"
	} else if strings.Contains(header, "webp") {
		ext = ".webp"
	} else {
		ext = ".jpg" // default
	}

	// Decode the base64 data
	imageData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %v", err)
	}

	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %v", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("group_post_%s_%d%s", postID, time.Now().Unix(), ext)
	filePath := filepath.Join(uploadsDir, filename)

	// Write the image data to file
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(imageData)
	if err != nil {
		return "", fmt.Errorf("failed to write image data: %v", err)
	}

	// Return just the filename (not the full path)
	return filename, nil
}

func GetGroupPosts(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query(`
		SELECT DISTINCT 
			p.id, 
			p.user_id, 
			u.username, 
			p.title, 
			p.content, 
			p.creation_date, 
			u.avatar,
			p.image
		FROM group_posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.group_id = $1
		ORDER BY p.creation_date DESC
	`, groupID)
	if err != nil {
		log.Println("[GetGroupPosts] DB query error:", err)
		http.Error(w, "Failed to get posts from database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts := make([]GroupPost, 0)
	for rows.Next() {
		var post GroupPost
		var imageFilename sql.NullString
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Author,
			&post.Title,
			&post.Content,
			&post.CreationDate,
			&post.Avatar,
			&imageFilename,
		)
		if err != nil {
			log.Println("[GetGroupPosts] Row scan error:", err)
			http.Error(w, "Failed to scan post row", http.StatusInternalServerError)
			return
		}
		if imageFilename.Valid && imageFilename.String != "" {
			post.Image = fmt.Sprintf("http://localhost:8080/uploads/%s", imageFilename.String)
		}
		posts = append(posts, post)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		log.Println("[GetGroupPosts] JSON encode error:", err)
		http.Error(w, "Failed to encode posts as JSON", http.StatusInternalServerError)
	}
}

// GetUserPendingInvitations gets all pending group invitations for a user
func GetUserPendingInvitations(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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

	// Query to get all groups where the user has pending invitations
	query := `
		SELECT g.id, g.title, g.description, u.username, gm.status
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		JOIN users u ON g.creator_id = u.id
		WHERE gm.user_id = $1 AND (gm.status = 'invited' OR gm.status = 'pending')
	`

	rows, err := db.DB.Query(query, userID)
	if err != nil {
		logger.LogError("Error fetching invitations", err)
		http.Error(w, "Error fetching invitations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Invitation struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		InvitedBy   string `json:"username"`
		Status      string `json:"status"`
	}

	var invitations []Invitation
	for rows.Next() {
		var invitation Invitation
		err := rows.Scan(&invitation.ID, &invitation.Title, &invitation.Description, &invitation.InvitedBy, &invitation.Status)
		if err != nil {
			logger.LogError("Error scanning invitation", err)
			continue
		}
		invitations = append(invitations, invitation)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}

// RequestToJoinGroup handles user requests to join a group
func RequestToJoinGroup(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get group ID from request body
	var request struct {
		GroupID string `json:"group_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.LogError("Invalid request", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
	// Validate group ID
	if request.GroupID == "" {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}
	// Check if the group exists
	var groupExists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)", request.GroupID).Scan(&groupExists)
	if err != nil {
		http.Error(w, "Failed to check group existence", http.StatusInternalServerError)
		return
	}
	// Check if user is already a member or has a pending request
	var exists bool
	var currentStatus string
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2), COALESCE((SELECT status FROM group_members WHERE group_id = $1 AND user_id = $2), '') as status",
		request.GroupID, userID).Scan(&exists, &currentStatus)
	if err != nil {
		logger.LogError("Error checking existing membership", err)
		http.Error(w, "Failed to check existing membership", http.StatusInternalServerError)
		return
	}

	if exists {
		if currentStatus == "invited" || currentStatus == "pending" {
			http.Error(w, fmt.Sprintf("User already has a %s request", currentStatus), http.StatusConflict)
			return
		}
		if currentStatus == "accepted" {
			http.Error(w, "User is already a member of the group", http.StatusConflict)
			return
		}
	}

	// Insert the join request with pending status
	query := "INSERT INTO group_members (group_id, user_id, is_admin, status) VALUES ($1, $2, '0', 'pending')"
	_, err = db.DB.Exec(query, request.GroupID, userID)
	if err != nil {
		logger.LogError("Failed to create join request", err)
		http.Error(w, "Failed to create join request", http.StatusInternalServerError)
		return
	}

	// Get group owner ID to send notification
	var ownerID string
	err = db.DB.QueryRow("SELECT creator_id FROM groups WHERE id = $1", request.GroupID).Scan(&ownerID)
	if err != nil {
		fmt.Println("Failed to get group owner ID:", err)
		http.Error(w, "Failed to get group owner ID", http.StatusInternalServerError)
		return
	}
	getFUllNameQuery := "SELECT first_name || ' ' || last_name as fullname FROM users WHERE id = $1"
	var ownerFullName string
	err = db.DB.QueryRow(getFUllNameQuery, userID).Scan(&ownerFullName)
	if err != nil {
		fmt.Println("Failed to get group owner username:", err)
		http.Error(w, "Failed to get group owner username", http.StatusInternalServerError)
		return
	}

	query1 := "select title from groups where id = $1"
	var groupName string
	err = db.DB.QueryRow(query1, request.GroupID).Scan(&groupName)
	if err != nil {
		fmt.Println("Failed to fetch group name", err)
		http.Error(w, "Failed to fetch group name", http.StatusInternalServerError)
		return
	}

	// Create notification for the group owner
	ownerUsername, _ := session.GetUsernameFromUserID(ownerID)
	userUsername, _ := session.GetUsernameFromUserID(userID)
	notification.CreateNotificationMessage(
		ownerUsername,
		userUsername,
		"GroupJoinRequest",
		ownerFullName+"has requested to join your group: "+groupName,
	)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Join request sent successfully"))
}

// GetGroupMemberStatuses returns the status of all users for a specific group
func GetGroupMemberStatuses(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get group ID from query parameters
	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Query to get all member statuses
	query := `
		SELECT user_id, status
		FROM group_members
		WHERE group_id = $1
	`

	rows, err := db.DB.Query(query, groupID)
	if err != nil {
		http.Error(w, "Error fetching member statuses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Map of user_id to status
	memberStatuses := make(map[string]string)
	for rows.Next() {
		var userID, status string
		err := rows.Scan(&userID, &status)
		if err != nil {
			http.Error(w, "Error scanning member status", http.StatusInternalServerError)
			return
		}
		memberStatuses[userID] = status
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memberStatuses)
}

// CancelGroupRequest handles canceling a pending group join request
func CancelGroupRequest(w http.ResponseWriter, r *http.Request) {
	// Handle CORS
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-rkwk3x6aq-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get group ID from request body
	var request struct {
		GroupID string `json:"group_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.LogError("Invalid request body", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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

	// Delete the pending request
	result, err := db.DB.Exec("DELETE FROM group_members WHERE group_id = $1 AND user_id = $2 AND status = 'pending'", request.GroupID, userID)
	if err != nil {
		logger.LogError("Failed to cancel group request", err)
		http.Error(w, "Failed to cancel group request", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to check if request was canceled", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "No pending request found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Group request canceled successfully"))
}
