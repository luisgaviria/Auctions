package utils

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var createUsersTable = `CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL UNIQUE,
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

var createFavoritesTable = `
CREATE TABLE IF NOT EXISTS favorites (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    auction_id INTEGER REFERENCES auctions(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, auction_id)
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

// addLastSeenColumn is idempotent: IF NOT EXISTS means it is safe to run on an
// existing database.  The UPDATE backfills rows that predate this migration so
// they are not swept on the very first scrape after deployment.
var addLastSeenColumn = `
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ;
UPDATE auctions SET last_seen = updated_at WHERE last_seen IS NULL;`

// addURLColumns adds the three generated URL columns introduced in the
// counties/Zillow/StreetView feature.  All three are nullable TEXT so that
// rows inserted before this migration (or rows where the city is unknown)
// are still valid.  The next scrape run will backfill them via ON CONFLICT.
var addURLColumns = `
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS zillow_url      TEXT;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS street_view_url TEXT;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS registry_url    TEXT;`

var addDeepLinkColumns = `
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS legal_description  TEXT;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS registry_deep_link TEXT;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS assessor_pid       TEXT;`

var addRegistryCoordinates = `
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS registry_book INTEGER;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS registry_page INTEGER;`

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

	_, err = db.Exec(createFavoritesTable)
	if err != nil {
		log.Fatal(err.Error())
		panic("Wrong Query For Favorites")
	}

	_, err = db.Exec(addLastSeenColumn)
	if err != nil {
		log.Fatal(err.Error())
		panic("Migration 006 failed: add last_seen column")
	}

	_, err = db.Exec(addURLColumns)
	if err != nil {
		log.Fatal(err.Error())
		panic("Migration 007 failed: add zillow_url / street_view_url / registry_url columns")
	}

	_, err = db.Exec(addDeepLinkColumns)
	if err != nil {
		log.Fatal(err.Error())
		panic("Migration 010 failed: add legal_description / registry_deep_link / assessor_pid columns")
	}

	_, err = db.Exec(addRegistryCoordinates)
	if err != nil {
		log.Fatal(err.Error())
		panic("Migration 011 failed: add registry_book / registry_page columns")
	}

	log.Print("Succesfully initialized tables!")
}
