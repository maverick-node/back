package posts

import (
	"fmt"
	"social-net/db"
)

func CheckUserPostPermission(userID string, postID string) bool {
	var (
		postOwnerID string
		status      string
	)

	err := db.DB.QueryRow(`SELECT user_id, status FROM posts WHERE id = $1`, postID).Scan(&postOwnerID, &status)
	if err != nil {
		fmt.Println("Error fetching post details 1:", err)
		return false // post makinchi
	}

	if userID == postOwnerID {
		return true // mol lpost howa requester
	}

	switch status {
	case "public":
		return true

	case "semi-private":
		// Check if user is in posts_privacy table
		var exists bool
		err := db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM posts_privacy
				WHERE post_id = $1 AND user_id = $2
			)`, postID, userID).Scan(&exists)
		if err != nil {
			fmt.Println("Error fetching post details 2:", err)
			return false
		}
		return exists

	case "private":
		// Check if user is a follower with accepted status
		var exists bool
		err := db.DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM Followers 
				WHERE follower_id = $1 AND followed_id = $2 AND status = 'accepted'
			)`, userID, postOwnerID).Scan(&exists)
		if err != nil {
			fmt.Println("Error fetching post details 3:", err)
			return false
		}
		return exists
	}

	return false
}
