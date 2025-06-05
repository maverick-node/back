package folowers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"social-net/db"
	logger "social-net/log"
	"social-net/notification"
	"social-net/session"

	"github.com/gofrs/uuid"
)

func SendJSON(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract session token and user ID
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		logger.LogError("Failed to get token", err)
		return
	}
	userID, ok := session.GetUserIDFromToken(token.Value)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	// Parse common query params

	action := r.URL.Query().Get("action")
	username := r.URL.Query().Get("profileUser")
	followerID := userID
	followedID, err := session.GetUserIDFromUsername(username)
	if err != nil {
		http.Error(w, "you can only follow existing users", http.StatusBadRequest)
		return
	}
	validActions := map[string]bool{
		"follow": true, "unfollow": true, "isFollowing": true,
		"getFollowing": true, "followersCount": true, "followingCount": true,
		"rejectInvitation": true,
	}
	if !validActions[action] {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	switch action {
	case "follow":
		UserToBeFollowed, err1 := session.GetUsernameFromUserID(followedID)
		if !err1 {
			fmt.Println("Error getting username:", err1)
			http.Error(w, "Error getting username", http.StatusInternalServerError)
			return
		}
		UserThatFollow, err2 := session.GetUsernameFromUserID(followerID)
		if !err2 {
			fmt.Println("Error getting username:", err2)
			http.Error(w, "Error getting username", http.StatusInternalServerError)
			return
		}
		if followerID == followedID {
			logger.LogError("You cannot follow yourself", nil)
			http.Error(w, "You cannot follow yourself", http.StatusBadRequest)
			return
		}
		status := "pending"

		roww := db.DB.QueryRow(`SELECT privacy FROM users WHERE id = $1`, followedID)
		var privacy string
		err = roww.Scan(&privacy)
		if err != nil {
			logger.LogError("Error getting privacy", err)
			http.Error(w, "Error getting privacy", http.StatusInternalServerError)
			return
		}
		getFUllNameQuery := `SELECT first_name, last_name FROM users WHERE id = $1`
		var firstName, lastName string
		err = db.DB.QueryRow(getFUllNameQuery, followerID).Scan(&firstName, &lastName)
		if err != nil {
			logger.LogError("Error getting full name", err)
			http.Error(w, "Error getting full name", http.StatusInternalServerError)
			return
		}

		if privacy == "private" {
			notification.CreateNotificationMessage(UserToBeFollowed, UserThatFollow, "follow_request", firstName+" "+lastName+" wants to follow you")
			fmt.Println(firstName + " " + lastName + " wants to follow you")
			status = "pending"
		} else {
			notification.CreateNotificationMessage(UserToBeFollowed, UserThatFollow, "follow_request", firstName+" "+lastName+" started following you")
			status = "accepted"
		}
		// Check if the follow request already exists
		var exists bool
		err = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM Followers WHERE follower_id = $1 AND followed_id = $2)`, followerID, followedID).Scan(&exists)
		if err != nil {
			http.Error(w, "Error checking follow request", http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, "You are already following this user", http.StatusBadRequest)
			return
		}
		followID, errr := uuid.NewV7()
		if errr != nil {
			http.Error(w, "Error generating follow ID", http.StatusInternalServerError)
			return
		}
		_, err := db.DB.Exec(`INSERT INTO Followers (id,follower_id, followed_id, status) VALUES ($1,$2, $3, $4)`, followID, followerID, followedID, status)
		if err != nil {
			logger.LogError("Error following user", err)
			http.Error(w, "Error following user", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case "unfollow":
		if followerID == followedID {
			http.Error(w, "You cannot unfollow yourself", http.StatusBadRequest)
			return
		}
		// Check if the follow request exists
		var exists bool
		err = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM Followers WHERE follower_id = $1 AND followed_id = $2)`, followerID, followedID).Scan(&exists)
		if err != nil {
			http.Error(w, "Error checking follow request", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "You are not following this user", http.StatusBadRequest)
			return
		}
		_, err := db.DB.Exec(`DELETE FROM Followers WHERE follower_id = $1 AND followed_id = $2`, followerID, followedID)
		if err != nil {
			logger.LogError("Error unfollowing user", err)
			http.Error(w, "Error unfollowing user", http.StatusInternalServerError)
			return
		}
		notification.DeleteNotification(followedID, followerID, "follow_request")
		w.WriteHeader(http.StatusOK)

	case "isFollowing":
		var exists bool
		err := db.DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM Followers 
				WHERE follower_id = $1 AND followed_id = $2 
			)`, followerID, followedID).Scan(&exists)
		if err != nil {
			logger.LogError("Error checking follow status", err)
			http.Error(w, "Error checking follow status", http.StatusInternalServerError)
			return
		}
		type isFollowing struct {
			IsFollowing bool   `json:"isFollowing"`
			Status      string `json:"status"`
		}
		var status string
		err1 := db.DB.QueryRow(`SELECT status FROM Followers WHERE follower_id = $1 AND followed_id = $2`, followerID, followedID).Scan(&status)
		if err1 == sql.ErrNoRows {
			json.NewEncoder(w).Encode(isFollowing{IsFollowing: false, Status: ""})
			return
		} else if err1 != nil {
			http.Error(w, "Error checking follow status", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(isFollowing{IsFollowing: true, Status: status})

	case "getFollowing":
		rows, err := db.DB.Query(`SELECT followed_id FROM Followers WHERE follower_id = $1 AND status = 'accepted'`, followerID)
		if err != nil {
			logger.LogError("Error getting following list", err)
			http.Error(w, "Error getting following list", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var following []int
		for rows.Next() {
			var fid int
			if err := rows.Scan(&fid); err != nil {
				logger.LogError("Error reading following list", err)
				http.Error(w, "Error reading following list", http.StatusInternalServerError)
				return
			}
			following = append(following, fid)
		}
		json.NewEncoder(w).Encode(following)

	case "followersCount":
		var count int
		err := db.DB.QueryRow(`SELECT COUNT(*) FROM Followers WHERE followed_id = $1 AND status = 'accepted'`, followedID).Scan(&count)
		if err != nil {
			logger.LogError("Error getting followers count", err)
			http.Error(w, "Error getting followers count", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]int{"followersCount": count})

	case "followingCount":
		var count int
		err := db.DB.QueryRow(`SELECT COUNT(*) FROM Followers WHERE follower_id = $1 AND status = 'accepted'`, followedID).Scan(&count)
		if err != nil {
			logger.LogError("Error getting following count", err)
			http.Error(w, "Error getting following count", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]int{"followingCount": count})

	case "rejectInvitation":

		// Delete the pending follow request
		_, err := db.DB.Exec(`DELETE FROM Followers WHERE follower_id = $1 AND followed_id = $2`, followedID, followerID)
		if err != nil {
			logger.LogError("Error rejecting invitation", err)
			http.Error(w, "Error rejecting invitation", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}
