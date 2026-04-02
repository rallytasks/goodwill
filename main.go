package main

import (
	"log"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "goodwill.db"
	}
	db := initDB(dbPath)
	defer db.Close()

	srv := newServer(db)

	log.Printf("Goodwill Donor Receipt App listening on :%s", port)
	log.Fatal(srv.ListenAndServe(":" + port))
}
