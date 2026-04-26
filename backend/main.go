package main

import (
	"backendAuction/config"
	"backendAuction/controllers"
	"backendAuction/middleware"
	"backendAuction/utils"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables first
	if os.Getenv("ENV") != "PROD" {
		if err := godotenv.Load(); err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	router := mux.NewRouter().StrictSlash(true)

	router.Use(middleware.CacheMiddleware)

	// CORS middleware — allows any origin listed in ALLOWED_ORIGINS env var.
	allowedOrigins := config.GetAllowedOrigins()
	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowedSet[o] = struct{}{}
	}
	log.Printf("[cors] allowed origins: %v", allowedOrigins)

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowedSet[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Cache-Control", "public, max-age=300")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	dbURL := config.GetDBURL()
	// Log only the host so credentials are never written to Railway/stdout logs.
	if parsed, err := url.Parse(dbURL); err == nil {
		log.Printf("[db] connecting to host=%s", parsed.Host)
	} else {
		log.Println("[db] connecting to database (url unparseable)")
	}

	db := utils.InitDb(dbURL)
	utils.InitTables(db)

	// Purge past auctions before each scrape run so stale records don't linger.
	if res, err := db.Exec("DELETE FROM auctions WHERE date IS NOT NULL AND date < (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date"); err != nil {
		log.Printf("[purge] failed to delete past auctions: %v", err)
	} else {
		n, _ := res.RowsAffected()
		log.Printf("[purge] deleted %d past auctions", n)
	}

	// utils.ScrapAllSites(db)

	// Initialize controllers
	authController := controllers.AuthController{DB: db}
	auctionController := controllers.AuctionsController{DB: db}
	favoritesController := controllers.FavoritesController{DB: db}
	scrapingController := controllers.ScrapingController{DB: db}

	// Auth routes
	authSubrouter := router.PathPrefix("/auth").Subrouter()
	authSubrouter.HandleFunc("/signup", authController.SignUp).Methods("POST", "OPTIONS")
	authSubrouter.HandleFunc("/login", authController.Login).Methods("POST", "OPTIONS")
	authSubrouter.HandleFunc("/logout", authController.Logout).Methods("POST", "OPTIONS")

	// Auctions routes
	auctionsSubrouter := router.PathPrefix("/auctions").Subrouter()
	auctionsSubrouter.HandleFunc("", auctionController.GetAuctions).Methods("GET", "OPTIONS")
	auctionsSubrouter.HandleFunc("/slugs", auctionController.GetTopSlugs).Methods("GET", "OPTIONS")
	auctionsSubrouter.HandleFunc("/{county_slug}/{city_slug}", auctionController.GetAuctionsBySlug).Methods("GET", "OPTIONS")

	// Report routes
	router.HandleFunc("/report/{address_slug}", auctionController.GetReport).Methods("GET", "OPTIONS")

	// Favorites routes
	favoritesSubrouter := router.PathPrefix("/favorites").Subrouter()
	favoritesSubrouter.HandleFunc("", middleware.AuthMiddleware(favoritesController.GetFavorites)).Methods("GET", "OPTIONS")
	favoritesSubrouter.HandleFunc("/add", middleware.AuthMiddleware(favoritesController.AddFavorite)).Methods("POST", "OPTIONS")
	favoritesSubrouter.HandleFunc("/remove", middleware.AuthMiddleware(favoritesController.RemoveFavorite)).Methods("POST", "OPTIONS")

	// Scraping routes
	scrapingSubrouter := router.PathPrefix("/scraping").Subrouter()
	scrapingSubrouter.HandleFunc("/start", scrapingController.StartScraping).Methods("POST", "OPTIONS")

	// Health check route
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	port := config.GetPort()

	srv := &http.Server{
		Handler:      router,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server is running on port %s", port)
	log.Fatal(srv.ListenAndServe())
}
