package main

import "log"

func logInfo(userID int64, username, message string) {
	log.Printf("[INFO] [User: %d | @%s] %s", userID, username, message)
}

func logError(userID int64, username, message string, err error) {
	log.Printf("[ERROR] [User: %d | @%s] %s: %v", userID, username, message, err)
}
