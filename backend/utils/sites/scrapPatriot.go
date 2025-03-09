package sites

import (
	"fmt"
	"regexp"
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

		// Parse and format the date
		formattedDate, formattedTime := parseDateAndTimePatriot(date)

		auctions = append(auctions, Auction{
			Status:  status,
			Street:  address,
			Url:     href,
			Date:    formattedDate,
			Time:    formattedTime,
			Deposit: deposit,
			Logo:    logo,
		})
		fmt.Println("patriot scrapping " + strconv.Itoa(i))
		page2.MustClose()
	}
	page.Browser().Close()
	return auctions
}

func parseDateAndTimePatriot(dateTimeStr string) (string, string) {
	// Regular expression to match "Monday Mar 10 @ 11:00 am"
	re := regexp.MustCompile(`(\w+ \w+ \d+) @ (\d+:\d+ [ap]m)`)
	match := re.FindStringSubmatch(dateTimeStr)

	if len(match) == 3 {
		dateStr := strings.TrimSpace(match[1])
		timeStr := strings.TrimSpace(match[2])
		parsedDate, err := time.Parse("Monday Jan 2", dateStr)
		if err != nil {
			fmt.Println("Error parsing date:", err)
			return "", timeStr
		}
		// Add the current year to the parsed date
		currentYear := time.Now().Year()
		parsedDate = parsedDate.AddDate(currentYear-parsedDate.Year(), 0, 0)
		formattedDate := parsedDate.Format("2006-01-02")
		return formattedDate, timeStr
	}
	return "", "" // Return empty if no match
}
