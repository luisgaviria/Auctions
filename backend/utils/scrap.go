package utils

import (
	"backendAuction/utils/sites"
	"database/sql"
	"fmt"
	"log"
)

var insertAuction = `INSERT INTO auctions (address, city, state, time, logo, status, link, date, deposit, lat, lng) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);`
var selectOneAuctionThroughAddress = `SELECT * FROM auctions WHERE address = $1`

func ScrapAllSites(db *sql.DB) {
	// auctions := sites.ScrapHarvard()
	// auctions := sites.ScrapJake()
	// auctions := sites.ScrapSri()
	// auctions := sites.ScrapSullivan()
	auctions := sites.ScrapAMG()
	// auctions = append(auctions, sites.ScrapCommon()...)
	// fmt.Println(sites.ScrapDanielP())
	// fmt.Println(sites.ScrapDanielP())
	// sites.ScrapApg()
	// sites.ScrapPatriot()
	// sites.ScrapBaystate()
	// fmt.Println(sites.ScrapDean())
	// fmt.Print(sites.ScrapTache())

	fmt.Println(auctions)
	for _, auction := range auctions {
		if auction, _ := db.Query(selectOneAuctionThroughAddress, auction.Street); auction != nil {
			log.Println("Auction exist!")
			continue
		}
		if _, err := db.Query(insertAuction, auction.Street, auction.City, "Massachusetts", auction.Time, auction.Logo, auction.Status, auction.Url, auction.Date, auction.Deposit, "0", "0"); err != nil {
			log.Println(err)
			return
		}
		log.Println("Placed auction!")
	}
}
