package utils

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var createUsersTable = `CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
	createdAt TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`

var createAuctionsTable = `CREATE TABLE IF NOT EXISTS auctions (
	id SERIAL PRIMARY KEY,
	address VARCHAR(255) NOT NULL,
    city VARCHAR(255),
	state VARCHAR(255),
	time VARCHAR(255), 
	logo VARCHAR(255),
	status VARCHAR(255) NOT NULL,
	link VARCHAR(255) NOT NULL,
	date DATE,
	deposit VARCHAR(255),
	lat VARCHAR(255) NOT NULL,
	lng VARCHAR(255) NOT NULL,
	createdAt TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`

func InitDb(url string) *sql.DB {
	db, err := sql.Open("postgres", url)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	log.Print("Succesfully connected to db")
	return db
}

func InitTables(db *sql.DB) {
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr.Error())
		panic("Wrong connection with DB")
	}
	_, err := db.Exec(createUsersTable)
	if err != nil {
		log.Fatal(err.Error())
		panic("Wrong Query For Users")
	}
	_, err = db.Exec(createAuctionsTable)
	if err != nil {
		log.Fatal(err.Error())
		panic("Wrong Query For Auctions")
	}
	log.Print("Succesfully initialized tables!")
}
