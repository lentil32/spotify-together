package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2"
)

type UserData struct {
	Token *oauth2.Token
}

func (db *Database) dumpToFile() {
	// Dangerous!
	fi, err := os.OpenFile(db.dbFile, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer fi.Close()

	dbData, _ := json.Marshal(db.userTable)
	if _, err := fi.Write(dbData); err != nil {
		log.Fatal(err)
	}
}

type Database struct {
	dbFile    string
	messages  chan uint8
	userTable map[string]*UserData
}

func newDatabase(dbFile string) *Database {
	var db *Database

	dat, err := os.ReadFile(dbFile)
	if err != nil {
		log.Printf("DB file `%s` does not exist.", dbFile)
		db = &Database{
			dbFile:    dbFile,
			messages:  make(chan uint8),
			userTable: make(map[string]*UserData),
		}
		return db
	}

	err = json.Unmarshal(dat, &db)
	if err != nil {
		removeErr := os.Remove(dbFile)
		if removeErr != nil {
			log.Fatal(err)
		}
		log.Fatal(err)
	}
	return &Database{
		dbFile:    dbFile,
		messages:  make(chan uint8),
		userTable: make(map[string]*UserData),
	}
}

func (db *Database) run() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case message := <-db.messages:
			log.Printf("DB Hub got message: %s", message)
		case t := <-ticker.C:
			log.Println("Dumped DB at: ", t)
			go db.dumpToFile()
		}
	}
}
