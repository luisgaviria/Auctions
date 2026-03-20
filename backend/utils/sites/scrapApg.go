package sites

import (
	"strings"

	"github.com/gocolly/colly/v2"
)

// ScrapApg scrapes the APG Online auction schedule.
//
// Each property card has <dd> elements in this order:
//   [0] Auction Status: ("Off" = off-site/off-market, always present)
//   [1] Property Status: ("Sold", "Active", …) — present only when status is set
//   [2] Address:
//   [3] Description:
//   [4] Required Deposit:
//
// When Property Status is absent the card has only 4 elements:
//   [0] Auction Status, [1] Address, [2] Description, [3] Deposit
//
// APG listings do not include a scheduled auction date.
func ScrapApg() []Auction {
	logo := "/apg.webp"
	url := "https://apg-online.com/auction-schedule/"

	c := colly.NewCollector()

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#content > div.columns.three.properties > div", func(i int, divElement *colly.HTMLElement) {
			ddTexts := divElement.ChildTexts("dd")

			var status, street, deposit string
			switch {
			case len(ddTexts) >= 5:
				// [0]=AuctionStatus [1]=PropertyStatus [2]=Address [3]=Desc [4]=Deposit
				status = ddTexts[1]
				street = strings.ReplaceAll(ddTexts[2], ".,", ",")
				deposit = ddTexts[len(ddTexts)-1]
			case len(ddTexts) == 4:
				// [0]=AuctionStatus [1]=Address [2]=Desc [3]=Deposit
				status = "Off Market"
				street = strings.ReplaceAll(ddTexts[1], ".,", ",")
				deposit = ddTexts[len(ddTexts)-1]
			default:
				return // not enough data
			}

			if street == "" {
				return
			}

			auctions = append(auctions, Auction{
				Status:  status,
				Street:  street,
				Deposit: deposit,
				Url:     url,
				Logo:    logo,
				// Date intentionally left empty — APG cards have no scheduled date.
			})
		})
	})

	c.Visit(url)

	return auctions
}
