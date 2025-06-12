package auth

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func Hashpwd(pswd string) string {
	pswdw, err := bcrypt.GenerateFromPassword([]byte(pswd), 10)
	if err != nil {
		return err.Error()
	}
	return string(pswdw)
}

func Validate(pswd string, pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(pswd), []byte(pass))
	return err == nil
}

func Senddata(w http.ResponseWriter, errCode int, mess string, data any) {
	response := struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	}{
		Error:   errCode,
		Message: mess,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus(errCode))
	json.NewEncoder(w).Encode(response)
}

func httpStatus(errCode int) int {
	switch errCode {
	case 0:
		return http.StatusOK
	case 1:
		return http.StatusBadRequest
	case 2:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
