package logger

import (
	"log"
	"os"
)

var (
	ErrorLogger *log.Logger
)

func init() {
	file, err := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open error log file:", err)
	}
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// LogError logs an error message to the error log file.
func LogError(message string, err error) {

	ErrorLogger.Printf("%s: %v", message, err)

}
