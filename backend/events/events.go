package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"social-net/db"
	"social-net/notification"
	"social-net/session"

	"github.com/gofrs/uuid"
)

type Event struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Date          string `json:"date"`
	Location      string `json:"location"`
	Response      *int   `json:"response"`
	GoingCount    int    `json:"going_count"`
	NotGoingCount int    `json:"not_going_count"`
}

type Event_Response struct {
	ID      string `json:"id"`
	UserID  int    `json:"user_id"`
	EventID int    `json:"event_id"`
	Option  int    `json:"option"`
}

const (
	ResponseGoing    = 1
	ResponseNotGoing = -1
)

type EventResponse struct {
	ID            string `json:"id"`
	EventID       string `json:"event_id"`
	Option        int    `json:"option"`
	GoingCount    int    `json:"going_count"`
	NotGoingCount int    `json:"not_going_count"`
}

func validateEventDate(dateStr string) error {
	eventDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {

		eventDate, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format: must be RFC3339 or YYYY-MM-DDThh:mm")
		}
	}

	now := time.Now()
	maxDate := now.AddDate(2, 0, 0)

	if eventDate.Before(now) {
		return fmt.Errorf("event date cannot be in the past")
	}

	if eventDate.After(maxDate) {
		return fmt.Errorf("event date cannot be more than 2 years in the future")
	}

	return nil
}

func sanitizeInput(input string) string {
	sanitized := strings.TrimSpace(input)
	return sanitized
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}

	token := cookie.Value
	userID, ok := session.GetUserIDFromToken(token)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Println("Error decoding request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	event.Title = sanitizeInput(event.Title)
	event.Description = sanitizeInput(event.Description)
	event.Location = sanitizeInput(event.Location)

	if event.Title == "" || event.Description == "" || event.Date == "" || event.Location == "" {
		log.Println("Missing required fields")
		http.Error(w, "Title, description, date, and location are required", http.StatusBadRequest)
		return
	}

	if len(event.Title) > 50 {
		http.Error(w, "Title must be less than 50 characters", http.StatusBadRequest)
		return
	}

	if len(event.Description) > 1000 {
		http.Error(w, "Description must be less than 1000 characters", http.StatusBadRequest)
		return
	}

	if len(event.Location) > 50 {
		http.Error(w, "Location must be less than 50 characters", http.StatusBadRequest)
		return
	}

	if err := validateEventDate(event.Date); err != nil {
		log.Printf("Date validation error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		log.Println("Group ID is required")
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM group_members WHERE user_id = $1 AND group_id = $2", userID, groupID).Scan(&count)
	if err != nil {
		log.Println("Error checking group membership:", err)
		http.Error(w, "Error checking group membership", http.StatusInternalServerError)
		return
	}
	if count == 0 {
		http.Error(w, "User is not a member of the group", http.StatusForbidden)
		return
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		log.Println("Error generating UUID:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	event.ID = eventID.String()

	query := `
		INSERT INTO events (id, creator_id, group_id, title, description, event_datetime, location)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err = db.DB.QueryRow(query, event.ID, userID, groupID, event.Title, event.Description, event.Date, event.Location).Scan(&event.ID)
	if err != nil {
		log.Println("Error inserting event:", err)
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	responseID, err := uuid.NewV7()
	if err != nil {
		log.Println("Error generating response UUID:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO event_responses (id, user_id, event_id, option)
		VALUES ($1, $2, $3, 1)`, responseID.String(), userID, event.ID)
	if err != nil {
		log.Println("Error setting creator response:", err)
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	go func() {
		rows, err := db.DB.Query(`
			SELECT user_id 
			FROM group_members 
			WHERE group_id = $1 AND user_id != $2`, groupID, userID)
		if err != nil {
			log.Printf("Error fetching group members for notifications: %v\n", err)
			return
		}
		defer rows.Close()

		userName, ok := session.GetUsernameFromUserID(userID)
		if !ok {
			log.Printf("Error getting username for user ID %s\n", userID)
			return
		}
		members := []string{}
		for rows.Next() {
			var memberID string
			if err := rows.Scan(&memberID); err != nil {
				log.Printf("Error scanning member ID: %v\n", err)
				continue
			}
			memberUsername, _ := session.GetUsernameFromUserID(memberID)
			members = append(members, memberUsername)
			fmt.Println("XXXXXXX1")

		}
		for _, member := range members {
			notification.CreateNotificationMessage(member, userName, "Event", "Event: "+event.Title+" has been created in your group.")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func JoinEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {
		log.Println("Missing event_id")
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	optionStr := r.URL.Query().Get("response")
	option, err := strconv.Atoi(optionStr)
	if err != nil || (option != ResponseGoing && option != ResponseNotGoing) {
		log.Printf("Invalid response option: %v\n", optionStr)
		http.Error(w, "Response must be either 'Going' (1) or 'Not Going' (-1)", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
		return
	}
	userID, ok := session.GetUserIDFromToken(cookie.Value)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var groupID string
	err = tx.QueryRow("SELECT group_id FROM events WHERE id = $1", eventID).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Event not found: %v\n", eventID)
			http.Error(w, "Event not found", http.StatusNotFound)
		} else {
			log.Printf("Database error: %v\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	var isMember bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM group_members WHERE user_id = $1 AND group_id = $2)",
		userID, groupID).Scan(&isMember)
	if err != nil {
		log.Printf("Error checking membership: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "You must be a group member to respond to events", http.StatusForbidden)
		return
	}

	var eventDate time.Time
	err = tx.QueryRow("SELECT event_datetime FROM events WHERE id = $1", eventID).Scan(&eventDate)
	if err != nil {
		log.Printf("Error getting event date: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if time.Now().After(eventDate) {
		http.Error(w, "Cannot respond to past events", http.StatusBadRequest)
		return
	}

	responseID, err := uuid.NewV7()
	if err != nil {
		log.Println("Error generating UUID:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	query := `
		INSERT INTO event_responses (id, user_id, event_id, option)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, event_id)
		DO UPDATE SET option = EXCLUDED.option
		RETURNING id`

	var responseRecordID string
	err = tx.QueryRow(query, responseID.String(), userID, eventID, option).Scan(&responseRecordID)
	if err != nil {
		log.Printf("Error updating response: %v\n", err)
		http.Error(w, "Failed to update response", http.StatusInternalServerError)
		return
	}

	var goingCount, notGoingCount int
	err = tx.QueryRow(`
		SELECT 
			COUNT(CASE WHEN option = 1 THEN 1 END) as going_count,
			COUNT(CASE WHEN option = -1 THEN 1 END) as not_going_count
		FROM event_responses 
		WHERE event_id = $1
	`, eventID).Scan(&goingCount, &notGoingCount)
	if err != nil {
		log.Printf("Error getting response counts: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := EventResponse{
		ID:            responseRecordID,
		EventID:       eventID,
		Option:        option,
		GoingCount:    goingCount,
		NotGoingCount: notGoingCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID := r.URL.Query().Get("id")
	if groupID == "" {
		log.Println("Group ID is required")
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		log.Println("Error getting token cookie:", err)
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	token := cookie.Value
	userID, ok := session.GetUserIDFromToken(token)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	query := `
	SELECT 
		e.id, 
		e.title, 
		e.description, 
		e.event_datetime, 
		e.location,
		er.option AS user_response,
		(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND option = 1) as going_count,
		(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND option = -1) as not_going_count
	FROM events e
	LEFT JOIN event_responses er 
		ON e.id = er.event_id AND er.user_id = ?
	WHERE e.group_id = ?
	ORDER BY e.event_datetime ASC
	`
	rows, err := db.DB.Query(query, userID, groupID)
	if err != nil {
		log.Println("Error fetching events:", err)
		http.Error(w, "Error fetching events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var (
			id, title, desc, datetime, location sql.NullString
			responseOption                      sql.NullInt64
			goingCount, notGoingCount           int
		)

		err := rows.Scan(&id, &title, &desc, &datetime, &location, &responseOption, &goingCount, &notGoingCount)
		if err != nil {
			log.Println("Error scanning event:", err)
			http.Error(w, "Error scanning event", http.StatusInternalServerError)
			return
		}

		if !id.Valid {
			log.Println("Invalid event with NULL ID encountered.")
			continue
		}

		event := Event{
			ID:            id.String,
			Title:         title.String,
			Description:   desc.String,
			Date:          datetime.String,
			Location:      location.String,
			GoingCount:    goingCount,
			NotGoingCount: notGoingCount,
		}

		if responseOption.Valid {
			val := int(responseOption.Int64)
			event.Response = &val
		}

		events = append(events, event)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
