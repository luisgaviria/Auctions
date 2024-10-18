package utils

import (
	"backendAuction/utils/sites"
	"fmt"
)

func ScrapAllSites() {
	auctions := sites.ScrapHarvard()
	fmt.Println(auctions)
}
