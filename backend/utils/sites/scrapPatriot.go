package sites

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

func ScrapPatriot() []Auction {
	auctions := make([]Auction, 0)
	logo := "/patriot.webp"
	page := rod.New().MustConnect().MustPage("https://patriotauctioneers.com/auctions-in-massachusetts/")
	page.MustWaitStable()

	div := page.MustElement("#calendar > div")
	as := div.MustElements("a")

	for i, a := range as {

		address := strings.TrimSpace(a.MustElement("h1").MustText())
		href := "https://patriotauctioneers.com" + *a.MustAttribute("href")
		fmt.Println(href)
		date := strings.TrimSpace(a.MustElement(".auction-date").MustText())

		date = strings.TrimSpace(strings.Split(date, "Continued")[0])

		page2 := page.Browser().MustPage(href)
		page2.MustWaitStable()

		deposit := strings.TrimSpace(strings.Split(page2.MustElement("#calendar > div:nth-child(2) > div > div.col-md-4 > div:nth-child(3) > p").MustText(), "deposit")[0])
		var status string
		err := rod.Try(func() {
			status = strings.TrimSpace(page2.Timeout(2 * time.Second).MustElement("#calendar > div:nth-child(2) > div > div.col-md-4 > div:nth-child(1) > p > span.text-red > strong").MustText())
		})

		if err != nil {
			status = "On Schedule"
		}

		auctions = append(auctions, Auction{
			Status:  status,
			Street:  address,
			Url:     href,
			Date:    date,
			Deposit: deposit,
			Logo:    logo,
		})
		fmt.Println("patriot scrapping " + strconv.Itoa(i))
		page2.MustClose()
	}
	page.Browser().Close()
	return auctions
}
