package utils

import (
	"backendAuction/utils/sites"
	"database/sql"
	"fmt"
	"log"
)

var insertAuction = `
    INSERT INTO auctions (address, city, state, time, logo, status, link, date, deposit, lat, lng) 
    VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
    RETURNING id;`

var selectOneAuctionThroughAddress = `SELECT * FROM auctions WHERE address = $1`

func ScrapAllSites(db *sql.DB) {
	auctions := sites.ScrapAMG()
	auctions = append(auctions, sites.ScrapApg()...)
	auctions = append(auctions, sites.ScrapBaystate()...)
	auctions = append(auctions, sites.ScrapAMG()...)
	auctions = append(auctions, sites.ScrapCommon()...)
	auctions = append(auctions, sites.ScrapDanielP()...)
	auctions = append(auctions, sites.ScrapDean()...)
	auctions = append(auctions, sites.ScrapHarvard()...)
	auctions = append(auctions, sites.ScrapJake()...)
	auctions = append(auctions, sites.ScrapPatriot()...)
	auctions = append(auctions, sites.ScrapSri()...)
	auctions = append(auctions, sites.ScrapSullivan()...)
	auctions = append(auctions, sites.ScrapTache()...)

	fmt.Println(auctions)
	for _, auction := range auctions {
		var exists int
		err := db.QueryRow(selectOneAuctionThroughAddress, auction.Street).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			log.Println(err)
			continue
		}
		if exists == 1 {
			log.Println("Auction exists!")
			continue
		}

		var auctionID int
		err = db.QueryRow(insertAuction,
			auction.Street,
			auction.City,
			"Massachusetts",
			auction.Time,
			auction.Logo,
			auction.Status,
			auction.Url,
			auction.Date,
			auction.Deposit,
			"0",
			"0").Scan(&auctionID)

		if err != nil {
			log.Print(auction)
			log.Println(err)
		} else {
			log.Printf("Placed auction with ID: %d!", auctionID)
		}
	}
}
