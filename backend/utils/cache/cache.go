package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var (
	// Create a cache with 5 minute default expiration and 10 minute cleanup interval
	Cache = cache.New(5*time.Minute, 10*time.Minute)
)

const (
	AuctionsKey = "all_auctions"
)
