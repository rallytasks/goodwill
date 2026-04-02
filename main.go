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

	db := initDB("goodwill.db")
	defer db.Close()

	srv := newServer(db)

	log.Printf("Goodwill Donor Receipt App listening on :%s", port)
	log.Fatal(srv.ListenAndServe(":" + port))
}
