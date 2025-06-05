package profile

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"social-net/db"
	logger "social-net/log"
	"social-net/session"
)

type UserInfo struct {
	Username           string   `json:"username"`
	Email              string   `json:"email"`
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	Bio                string   `json:"bio"`
	DateOfBirth        string   `json:"date_of_birth"`
	Privacy            string   `json:"privacy"`
	FollowersCount     int      `json:"followers_count"`
	FollowingCount     int      `json:"following_count"`
	FollowerUsernames  []string `json:"follower_usernames"`
	FollowingUsernames []string `json:"following_usernames"`
	PostsCount         int      `json:"posts"`
	Avatar             string   `json:"avatar"`
	Nickname           string   `json:"nickname"`
	FollowStatus       string   `json:"follow_status"` // Added follow status
}

type GetPost struct {
	Id            string `json:"id"`
	User_id       string `json:"user_id"`
	Author        string `json:"author"`
	Content       string `json:"content"`
	Title         string `json:"title"`
	Creation_date string `json:"creation_date"`
	Status        string `json:"status"`
	Avatar        string `json:"avatar"`
	Image         string `json:"image"`
	CommentsCount int    `json:"comments_count"`
}

type Comments struct {
	Id        string `json:"id"`
	PostId    string `json:"post_id"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	token, err := r.Cookie("token")
	if token == nil || err != nil {
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	sessionToken := token.Value
	user_id, ok := session.GetUserIDFromToken(sessionToken)
	if !ok || user_id == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username := r.URL.Query().Get("user_id")
	if username == "" {
		logger.LogError("Username is required", nil)
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// First, get the user's ID from the username
	var userID string
	err1 := db.DB.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err1 != nil {
		if err1 == sql.ErrNoRows {
			fmt.Println("No user found with the given username")
			http.Error(w, "No user found", http.StatusNotFound)
			return
		}
		logger.LogError("Error fetching user ID", err)
		http.Error(w, "Error fetching user ID", http.StatusInternalServerError)
		return
	}

	var userInfo UserInfo
	err = db.DB.QueryRow("SELECT username, email, first_name, last_name, bio, date_of_birth, privacy, avatar, nickname FROM users WHERE id = $1", userID).Scan(
		&userInfo.Username, &userInfo.Email, &userInfo.FirstName, &userInfo.LastName, &userInfo.Bio, &userInfo.DateOfBirth, &userInfo.Privacy, &userInfo.Avatar, &userInfo.Nickname)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.LogError("No user found with the given ID", nil)
			http.Error(w, "No user found", http.StatusNotFound)
			return
		}
		fmt.Println("Error fetching user info:", err)
		http.Error(w, "Error fetching user info", http.StatusInternalServerError)
		return
	}
	// Check if the current user is following the profile user
	query := `
		SELECT status FROM followers WHERE follower_id = $1 AND followed_id = $2`
	var followStatus string
	err = db.DB.QueryRow(query, user_id, userID).Scan(&followStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			followStatus = "not_following" // User is not following
		} else {
			logger.LogError("Error checking follow status", err)
			http.Error(w, "Error checking follow status", http.StatusInternalServerError)
			return
		}
	}
	userInfo.FollowStatus = followStatus
	followersCount, err := GetFollowersCount(userID)
	if err != nil {
		logger.LogError("Error getting followers count", err)
		http.Error(w, "Error getting followers count", http.StatusInternalServerError)
		return
	}

	followingCount, err := GetFollowingCount(userID)
	if err != nil {
		logger.LogError("Error getting following count", err)
		http.Error(w, "Error getting following count", http.StatusInternalServerError)
		return
	}

	followers, err := GetFollowerUsernames(userID)
	if err != nil {
		logger.LogError("Error getting follower usernames", err)
		http.Error(w, "Error getting follower usernames", http.StatusInternalServerError)
		return
	}

	following, err := GetFollowingUsernames(userID)
	if err != nil {
		logger.LogError("Error getting following usernames", err)
		http.Error(w, "Error getting following usernames", http.StatusInternalServerError)
		return
	}

	var postCount int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = $1", userID).Scan(&postCount)
	if err != nil {
		logger.LogError("Error counting posts", err)
		http.Error(w, "Error getting post count", http.StatusInternalServerError)
		return
	}

	userInfo.FollowersCount = followersCount
	userInfo.FollowingCount = followingCount
	userInfo.FollowerUsernames = followers
	userInfo.FollowingUsernames = following
	userInfo.PostsCount = postCount

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

func UpdatePrivacy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var request struct {
		Privacy string `json:"privacy"`
	}
	// Get the user ID from the session
	token, err := r.Cookie("token")
	fmt.Println("TokenVAlue:", token.Value)
	fmt.Println("Token:", token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := session.GetUserIDFromToken(token.Value)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the privacy value
	if request.Privacy != "public" && request.Privacy != "private" {
		http.Error(w, "Invalid privacy value", http.StatusBadRequest)
		return
	}

	username, ok1 := session.GetUsernameFromUserID(userID)
	if !ok1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Update the privacy setting in the database
	_, err = db.DB.Exec("UPDATE users SET privacy = $1 WHERE username = $2", request.Privacy, username)
	if err != nil {
		logger.LogError("Failed to update privacy", err)
		http.Error(w, "Failed to update privacy", http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func GetFollowersCount(userID string) (int, error) {
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM followers WHERE followed_id = $1 AND status = 'accepted'", userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetFollowingCount(userID string) (int, error) {
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM followers WHERE follower_id = $1 AND status = 'accepted'", userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetFollowerUsernames(userID string) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT u.username 
		FROM users u 
		JOIN followers f ON u.id = f.follower_id 
		WHERE f.followed_id = $1 AND f.status = 'accepted'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		usernames = append(usernames, username)
	}
	return usernames, nil
}

func GetFollowingUsernames(userID string) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT u.username 
		FROM users u 
		JOIN followers f ON u.id = f.followed_id 
		WHERE f.follower_id = $1 AND f.status = 'accepted'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		usernames = append(usernames, username)
	}
	return usernames, nil
}

func IsFollowing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	sessionToken := token.Value
	userID, ok := session.GetUserIDFromToken(sessionToken)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	followerUsername := r.URL.Query().Get("follower_id")
	followedUsername := r.URL.Query().Get("followed_id")

	log.Println("Follower Username:", followerUsername)
	log.Println("Followed Username:", followedUsername)

	if followerUsername == "" || followedUsername == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// Get the user IDs from the usernames
	var followerID, followedID string
	err = db.DB.QueryRow("SELECT id FROM users WHERE username = $1", followerUsername).Scan(&followerID)
	if err != nil {
		http.Error(w, "Error finding follower user", http.StatusInternalServerError)
		return
	}

	err = db.DB.QueryRow("SELECT id FROM users WHERE username = $1", followedUsername).Scan(&followedID)
	if err != nil {
		http.Error(w, "Error finding followed user", http.StatusInternalServerError)
		return
	}

	// Check if the follower is following the followed user
	var exists bool
	err = db.DB.QueryRow(`SELECT EXISTS(
        SELECT 1 FROM followers 
        WHERE follower_id = $1 AND followed_id = $2 AND status = 'accepted'
    )`, followerID, followedID).Scan(&exists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"isFollowing": exists})
}

func IsAcceptedFollower(viewerID, ownerID string) (bool, error) {
	var exists bool
	err := db.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = $1 AND followed_id = $2 AND status = 'accepted')",
		viewerID, ownerID,
	).Scan(&exists)
	return exists, err
}

// In getposts.go
func GetOwnPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	token, err1 := r.Cookie("token")
	if err1 != nil {
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	sessionToken := token.Value
	CurrentUserid, ok := session.GetUserIDFromToken(sessionToken)
	if !ok || CurrentUserid == "" {
		fmt.Println("Invalid token")
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username := r.URL.Query().Get("username")

	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var userID string
	err := db.DB.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No user found with the given username")
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		fmt.Println("Error fetching user ID:", err)
		http.Error(w, "Error finding user", http.StatusInternalServerError)
		return
	}
	query := `
		SELECT DISTINCT p.id, p.user_id, p.author, p.content, p.title, p.creation_date, p.status, u.avatar, p.image
		FROM posts p
		LEFT JOIN posts_privacy pp ON p.id = pp.post_id
		LEFT JOIN users u ON p.user_id = u.id
		WHERE p.user_id = $1
  		AND (
    	p.status = 'public'
    	OR (p.status = 'private' AND EXISTS (
        SELECT 1 FROM followers WHERE follower_id = $2 AND followed_id = $3 AND status = 'accepted'
    	))
    	OR (p.status = 'semi-private' AND pp.user_id = $4)
    	OR ($5 = p.user_id)
  )
ORDER BY p.creation_date DESC
	`
	rows, err := db.DB.Query(query, userID, CurrentUserid, userID, CurrentUserid, CurrentUserid)
	if err != nil {
		http.Error(w, "Error querying posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []GetPost
	for rows.Next() {
		var post GetPost
		err := rows.Scan(&post.Id, &post.User_id, &post.Author, &post.Content, &post.Title, &post.Creation_date, &post.Status, &post.Avatar, &post.Image)
		if err != nil {
			http.Error(w, "Error scanning posts", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	for i, post := range posts {
		// Get the comments count for each post
		var commentsCount int
		err := db.DB.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = $1", post.Id).Scan(&commentsCount)
		if err != nil {
			fmt.Println("Error getting comments count:", err)
			http.Error(w, "Error getting comments count", http.StatusInternalServerError)
			return
		}
		posts[i].CommentsCount = commentsCount
	}
	// Always return JSON (even if empty)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func GetFollowersAndFollowing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get the profile user from query parameter
	profileUser := r.URL.Query().Get("profileUser")
	if profileUser == "" {
		http.Error(w, "Profile user is required", http.StatusBadRequest)
		return
	}

	// Get the current user from session
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tokenValue := token.Value
	userID, ok := session.GetUserIDFromToken(tokenValue)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	currentUser, _ := session.GetUsernameFromUserID(userID)

	// Query to get followers
	followersQuery := `
        SELECT follower_id
        FROM followers
        WHERE followed_id = (SELECT id FROM users WHERE username = $1) and status = 'accepted'
    `
	followersRows, err := db.DB.Query(followersQuery, profileUser)
	if err != nil {
		fmt.Println("err2", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer followersRows.Close()

	var followers []string
	for followersRows.Next() {
		var followerID string
		err := followersRows.Scan(&followerID)
		if err != nil {
			continue
		}
		// Get username from user ID
		username, _ := session.GetUsernameFromUserID(userID)
		followers = append(followers, username)
	}

	// Query to get following
	followingQuery := `
        SELECT followed_id
        FROM followers
        WHERE follower_id = (SELECT id FROM users WHERE username = $1) and status='accepted'
    `
	followingRows, err := db.DB.Query(followingQuery, profileUser)
	if err != nil {
		fmt.Println("err1", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer followingRows.Close()

	var following []string
	for followingRows.Next() {
		var followingID string
		err := followingRows.Scan(&followingID)
		if err != nil {
			continue
		}
		// Get username from user ID
		username, _ := session.GetUsernameFromUserID(followingID)
		following = append(following, username)
	}

	// Prepare response
	response := map[string]interface{}{
		"followers":      followers,
		"following":      following,
		"is_own_profile": currentUser == profileUser,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetFollowersAndFollowingPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get the profile user from query parameter
	// profileUser := r.URL.Query().Get("profileUser")
	// if profileUser == "" {
	// 	http.Error(w, "Profile user is required", http.StatusBadRequest)
	// 	return
	// }

	// Get the current user from session
	token, _ := r.Cookie("token")
	tokenValue := token.Value
	userID, ok := session.GetUserIDFromToken(tokenValue)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	// currentUser, _ := session.Getusernamefromuserid(userID)

	// Query to get followers
	followersQuery := `
		SELECT follower_id
		FROM followers
		WHERE followed_id = $1 and status = 'accepted'
    `
	followersRows, err := db.DB.Query(followersQuery, userID)
	if err != nil {
		logger.LogError("Database error", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer followersRows.Close()

	var followers []string
	for followersRows.Next() {
		var followerID string
		err := followersRows.Scan(&followerID)
		if err != nil {
			logger.LogError("Error scanning follower ID", err)
			continue
		}
		// Get username from user ID
		username, _ := session.GetUsernameFromUserID(followerID)
		followers = append(followers, username)
	}

	// Query to get following
	followingQuery := `
        SELECT followed_id
        FROM followers
        WHERE follower_id = (SELECT id FROM users WHERE username = $1) and status='accepted'
    `
	followingRows, err := db.DB.Query(followingQuery, userID)
	if err != nil {
		logger.LogError("Database error", err)

		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer followingRows.Close()

	var following []string
	for followingRows.Next() {
		var followingID string
		err := followingRows.Scan(&followingID)
		if err != nil {
			logger.LogError("Error scanning following ID", err)
			continue
		}
		// Get username from user ID
		username, _ := session.GetUsernameFromUserID(followingID)
		following = append(following, username)
	}

	// Prepare response
	response := map[string]interface{}{
		"followers": followers,
		"following": following,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func CheckMyPrivacy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	token, _ := r.Cookie("token")
	username1, ok := session.GetUserIDFromToken(token.Value)
	if !ok || username1 == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username, _ := session.GetUsernameFromUserID(username1)

	row := db.DB.QueryRow(`SELECT privacy FROM users WHERE username=$1`, username)
	var privacy string
	err := row.Scan(&privacy)
	if err != nil {
		logger.LogError("Failed to fetch privacy", err)
		http.Error(w, "Failed to fetch privacy", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"privacy": privacy})
}

func GetInvitationsFollow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get the user ID from the request context
	token, _ := r.Cookie("token")
	tokenValue := token.Value
	userID, ok := session.GetUserIDFromToken(tokenValue)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Fetch the invitations from the database
	rows, err := db.DB.Query("SELECT follower_id FROM Followers WHERE followed_id = $1 AND status = 'pending'", userID)
	if err != nil {
		fmt.Println("Error fetching invitations:", err)
		http.Error(w, "Failed to fetch invitations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var invitations []struct {
		FollowerID string `json:"follower_id"`
		Username   string `json:"username"`
	}
	for rows.Next() {
		var invitation struct {
			FollowerID string `json:"follower_id"`
			Username   string `json:"username"`
		}
		err := rows.Scan(&invitation.FollowerID)
		if err != nil {
			fmt.Println("Error scanning invitation:", err)
			http.Error(w, "Failed to scan invitation", http.StatusInternalServerError)
			return
		}

		// Fetch the username for the follower ID
		err = db.DB.QueryRow("SELECT username FROM users WHERE id = $1", invitation.FollowerID).Scan(&invitation.Username)
		if err != nil {
			fmt.Println("Error fetching username:", err)
			http.Error(w, "Failed to fetch username", http.StatusInternalServerError)
			return
		}

		invitations = append(invitations, invitation)
	}

	fmt.Println("invitations", invitations)

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}

func AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get the user ID from the request context
	token, _ := r.Cookie("token")
	tokenValue := token.Value
	userID, ok := session.GetUserIDFromToken(tokenValue)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Get the follower ID from the request body
	var data struct {
		FollowerID string `json:"follower_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	fmt.Println("data", data)

	follower_id, _ := session.GetUserIDFromUsername(data.FollowerID)

	if err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	// Update the status in the database
	_, err = db.DB.Exec("UPDATE Followers SET status = 'accepted' WHERE follower_id = $1 AND followed_id = $2", follower_id, userID)
	if err != nil {
		fmt.Println("Error updating invitation status:", err)
		http.Error(w, "Failed to update invitation status", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
