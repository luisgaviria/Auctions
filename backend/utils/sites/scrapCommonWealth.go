package sites

import (
	"log"

	"github.com/go-rod/rod"
)

func scrapTable(page *rod.Page) []Auction {
	auctions := make([]Auction, 0)
	tbody := page.MustElement("#ma_auctions > tbody")
	trOdds, err := tbody.Elements("tr.odd")
	trEvens, err := tbody.Elements("tr.even")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	for _, tr := range trOdds {
		tds, _ := tr.Elements("td")

		url := tds[5].MustElement("a").MustAttribute("href")
		auctions = append(auctions, Auction{
			Status:  tds[3].MustText(),
			Logo:    "/commonwealth.webp",
			Date:    tds[0].MustText(),
			Street:  tds[1].MustText(),
			City:    tds[2].MustText(),
			Deposit: tds[4].MustText(),
			Url:     *url,
		})
	}

	for _, tr := range trEvens {
		tds, _ := tr.Elements("td")

		url := tds[5].MustElement("a").MustAttribute("href")
		auctions = append(auctions, Auction{
			Status:  tds[3].MustText(),
			Logo:    "/commonwealth.webp",
			Date:    tds[0].MustText(),
			Street:  tds[1].MustText(),
			City:    tds[2].MustText(),
			Deposit: tds[4].MustText(),
			Url:     *url,
		})
	}
	return auctions
}

func ScrapCommon() []Auction {
	auctions := make([]Auction, 0)
	page := rod.New().MustConnect().MustPage("https://www.commonwealthauctions.com/ma-auctions")
	page.MustWaitStable()

	auctions = scrapTable(page)

	nextButton := page.MustElement("#ma_auctions_next")
	nextButton.MustClick()

	auctions = append(auctions, scrapTable(page)...)

	page.Browser().Close()

	return auctions
}
