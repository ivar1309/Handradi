package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ivar1309/Handradi/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cli [add|list|delete] ...")
		return
	}

	db.InitDB()
	defer db.Close()

	cmd := os.Args[1]

	switch cmd {
	case "add":
		if len(os.Args) != 5 {
			fmt.Println("Usage: cli add <client_id> <api_key> <allowed_origin>")
			return
		}
		clientID := os.Args[2]
		apiKey := os.Args[3]
		origin := os.Args[4]
		err := db.AddUser(clientID, apiKey, origin)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("✅ Client added:", clientID)

	case "list":
		users, err := db.AllUsers()
		if err != nil {
			log.Fatal(err)
		}

		for _, user := range users {
			fmt.Printf("User: %v\t%v\t%v\n", user.ClientId, user.ApiKey, user.AllowedOrigin)
		}

	case "delete":
		if len(os.Args) != 3 {
			fmt.Println("Usage: cli delete <client_id>")
			return
		}
		clientID := os.Args[2]
		err := db.DeleteUser(clientID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("🗑️ Client deleted:", clientID)

	default:
		fmt.Println("Unknown command:", cmd)
	}
}
