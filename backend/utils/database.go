package utils

import (
	"database/sql"
	"log"

	"backendAuction/migrations"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

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

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	// For existing databases without goose tracking, backfill the version to 11
	// so we don't try to re-run historical raw migrations that lack IF NOT EXISTS clauses.
	var gooseExists bool
	db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'goose_db_version')").Scan(&gooseExists)
	
	if !gooseExists {
		var auctionsExists bool
		db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'auctions')").Scan(&auctionsExists)
		if auctionsExists {
			log.Print("Existing database detected. Backfilling goose version to 11...")
			_, err := db.Exec(`CREATE TABLE goose_db_version (
				id serial PRIMARY KEY,
				version_id bigint NOT NULL,
				is_applied boolean NOT NULL,
				tstamp timestamp without time zone DEFAULT now()
			)`)
			if err == nil {
				for i := 1; i <= 11; i++ {
					db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)`, i)
				}
			}
		}
	} else {
		// Backfill 1-10 if they are missing (fix for previous partial backfill)
		var count int
		db.QueryRow("SELECT COUNT(*) FROM goose_db_version").Scan(&count)
		if count == 1 {
			for i := 1; i <= 10; i++ {
				db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)`, i)
			}
		}
	}

	if err := goose.Up(db, "."); err != nil {
		log.Fatalf("goose up error: %v", err)
	}

	log.Print("Succesfully initialized tables via goose migrations!")
}
