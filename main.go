package main

import (
	"fmt"
)

func main() {
	apiKey := "your_api_key"
	apiSecret := "your_api_secret"
	sessionToken := "your_session_token"

	bc := NewBreezeConnect(apiKey)
	if err := bc.generateSession(apiSecret, sessionToken); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Session generated successfully")
	// Add more functionalities here...
}
