package sites

import (
	"log"
	"strings"

	"github.com/gocolly/colly"
)

// ScrapeAuctionDetails scrapes a single auction detail page.
func scrapAuctionDetails(baseURL, detailURL string, auction *Auction) error { // Modified signature
	// Construct the absolute URL.  This is *CRITICAL*.
	absoluteURL := baseURL + detailURL
	auction.Url = absoluteURL

	c := colly.NewCollector(
		colly.AllowedDomains("www.amgauction.com"),
	)

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed:", err)
	})

	// Extract Date and Time.
	c.OnHTML("time.event-date", func(e *colly.HTMLElement) {
		auction.Date = e.Attr("datetime")
	})
	c.OnHTML("time.event-time-localized-start", func(e *colly.HTMLElement) {
		auction.Time = e.Text
	})

	// Extract Address.  This is more robust than relying on .eventitem-meta-address-line
	c.OnHTML(".eventitem-meta-address", func(e *colly.HTMLElement) {
		var addressLines []string
		e.ForEach(".eventitem-meta-address-line", func(_ int, el *colly.HTMLElement) {
			addressLines = append(addressLines, strings.TrimSpace(el.Text)) //Clean up the line
		})

		fullAddress := strings.Join(addressLines, ", ") // combine the lines

		parts := strings.Split(fullAddress, ",") //split address parts
		if len(parts) >= 2 {
			auction.Street = strings.TrimSpace(parts[0]) //first part of address is street

			cityState := strings.TrimSpace(parts[1]) // second part is city, state
			cityStateParts := strings.Split(cityState, " ")
			if len(cityStateParts) > 0 {
				auction.City = strings.TrimSpace(cityStateParts[0]) //first part is city
			}

		}

	})

	// Extract Property Details and deposit,  This is much more robust
	c.OnHTML(".sqs-html-content", func(e *colly.HTMLElement) { //select the div containing that info

		e.ForEach("p", func(_ int, el *colly.HTMLElement) { //iterate <p> tags inside it
			text := strings.TrimSpace(el.Text) //get text

			// Skip any empty lines
			if text == "" {
				return
			}
			//Special case for Deposit
			if strings.Contains(text, "Terms:") && strings.Contains(text, "$") {
				parts := strings.SplitN(text, "$", 2) //split by $
				if len(parts) > 1 {
					depositPart := strings.SplitN(parts[1], " ", 2)[0] // separate the number
					auction.Deposit = "$" + depositPart
					text = strings.ReplaceAll(text, "$"+depositPart, "") //Remove it
				}
			}

		})
	})

	// Extract first image as logo
	c.OnHTML(".sqs-gallery-block-slideshow img.thumb-image", func(e *colly.HTMLElement) {
		if auction.Logo == "" { // Take ONLY THE FIRST IMAGE
			auction.Logo = "/amg.webp" // Use a default logo
		}
	})

	err := c.Visit(absoluteURL) // Use the *absolute* URL here.
	if err != nil {
		return err // IMPORTANT: Return the error if the visit fails.
	}
	return nil
}

// ScrapeAMG scrapes the main auction listing page and then each individual auction page.
func ScrapAMG() []Auction {
	c := colly.NewCollector(
		colly.AllowedDomains("www.amgauction.com"),
	)

	baseURL := "https://www.amgauction.com" // Define the base URL
	auctions := []Auction{}

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed:", err)
	})

	c.OnHTML("article.eventlist-event", func(e *colly.HTMLElement) {
		auctionURL := e.ChildAttr("a.eventlist-title-link", "href")
		if auctionURL != "" {
			auction := Auction{}
			// Extract the Status
			if strings.Contains(e.Attr("class"), "eventlist-event--past") {
				auction.Status = "Past"
			} else {
				auction.Status = "Upcoming" // Or "Current", as appropriate
			}
			//get auction detail
			err := scrapAuctionDetails(baseURL, auctionURL, &auction) // Pass base url
			if err != nil {
				log.Println("Error scraping details:", err)
			} else {
				auctions = append(auctions, auction) //add auction to list
			}
		}
	})

	err := c.Visit("https://www.amgauction.com/auctions") //starts the scraping
	if err != nil {
		return nil
	}

	return auctions
}
