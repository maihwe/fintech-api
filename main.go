package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "fintech.db"
	}
	initDB(dbPath)
	defer db.Close()

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/balance", requireAuth(balanceHandler))
	http.HandleFunc("/deposit", requireAuth(depositHandler))
	http.HandleFunc("/withdraw", requireAuth(withdrawHandler))
	http.HandleFunc("/transfer", requireAuth(transferHandler))
	http.HandleFunc("/transactions", requireAuth(transactionsHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Fintech API running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}