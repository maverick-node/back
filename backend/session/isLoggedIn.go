package session

import "net/http"

func IsLoggedIn(r *http.Request) bool {
	tokene, err := r.Cookie("token")
	if err != nil {
		return false
	}
	token := tokene.Value
	userID, err1 := GetUserIDFromToken(token)
	if err1 {
		return false
	}
	if userID == "" {
		return false
	}
	return true
}
