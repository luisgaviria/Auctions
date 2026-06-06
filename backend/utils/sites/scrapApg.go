package sites

import (
	"strings"
	"time"

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
	c.SetRequestTimeout(60 * time.Second)

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#content > div.columns.three.properties > div", func(i int, divElement *colly.HTMLElement) {
			ddTexts := divElement.ChildTexts("dd")

			if len(ddTexts) < 4 {
				return // not enough data
			}

			// ddTexts[0] is always Auction Status — skip anything that is not active.
			auctionStatus := strings.TrimSpace(ddTexts[0])
			if strings.EqualFold(auctionStatus, "off") {
				return
			}

			var status, street, date, deposit, legalDesc string
			switch {
			case len(ddTexts) >= 5:
				// [0]=AuctionStatus [1]=PropertyStatus [2]=Address [3]=Desc [4]=Deposit
				status = ddTexts[1]
				street = strings.ReplaceAll(ddTexts[2], ".,", ",")
				legalDesc = strings.TrimSpace(ddTexts[3])
				deposit = ddTexts[len(ddTexts)-1]
			default:
				// [0]=AuctionStatus [1]=Address [2]=Desc [3]=Deposit
				status = auctionStatus
				street = strings.ReplaceAll(ddTexts[1], ".,", ",")
				legalDesc = strings.TrimSpace(ddTexts[2])
				deposit = ddTexts[len(ddTexts)-1]
			}

			// Extract date from <dt>/<dd> pairs by label so we never rely on position.
			divElement.ForEach("dl.custom_field_list dt", func(_ int, dt *colly.HTMLElement) {
				label := strings.ToLower(strings.TrimSpace(dt.Text))
				if strings.Contains(label, "date") || strings.Contains(label, "auction date") {
					dd := dt.DOM.Next()
					if dd != nil {
						date = strings.TrimSpace(dd.Text())
					}
				}
			})

			if street == "" || date == "" {
				return // skip if no address or no scheduled date
			}

			auctions = append(auctions, Auction{
				Status:           status,
				Street:           street,
				Date:             date,
				Deposit:          deposit,
				Url:              url,
				Logo:             logo,
				LegalDescription: legalDesc,
			})
		})
	})

	c.Visit(url)

	return auctions
}
