package main

import (
	"context"
	"flag"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
	"log"
	"naimuBack/internal/config"
	"naimuBack/internal/taxi"
	"net/http"
	"os"
	"time"
)

type taxiLogger struct{ info, err *log.Logger }

func (l taxiLogger) Infof(f string, a ...interface{})  { l.info.Printf(f, a...) }
func (l taxiLogger) Errorf(f string, a ...interface{}) { l.err.Printf(f, a...) }

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	cfg := config.LoadConfig()

	port := os.Getenv("PORT")
	if port == "" {
		port = ":4001"
	} else {
		port = ":" + port
	}

	addr := flag.String("addr", port, "HTTP network address")
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	if os.Getenv("OPENAI_API_KEY") == "" {
		infoLog.Println("OpenAI: DISABLED (no OPENAI_API_KEY)")
	} else {
		infoLog.Println("OpenAI: ENABLED")
	}

	db, err := openDB(cfg.Database.URL)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	app := initializeApp(db, errorLog, infoLog)

	// === Redis (боевой) ===
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
		DB:   0,
	})
	defer func() {
		if err := rdb.Close(); err != nil {
			errorLog.Printf("failed to close redis: %v", err)
		}
	}()

	// === Конфиг Taxi из ENV (ошибка — фатал)
	taxiCfg, err := taxi.LoadTaxiConfig()
	if err != nil {
		errorLog.Fatal(err)
	}

	// === Собираем зависимости Taxi
	deps := &taxi.TaxiDeps{
		DB:         db,
		RDB:        rdb,
		Logger:     taxiLogger{infoLog, errorLog},
		Config:     taxiCfg,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	// === Заводим stdlib mux для такси и регистрируем его маршруты
	taxiMux := http.NewServeMux()
	if err := taxi.RegisterTaxiRoutes(taxiMux, deps); err != nil {
		errorLog.Fatal(err)
	}

	// === Прокидываем в app, чтобы routes() мог смонтировать такси-маршруты
	app.taxiMux = taxiMux
	app.taxiDeps = deps

	// === Запускаем фоновые воркеры такси
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := taxi.StartTaxiWorkers(ctx, deps); err != nil {
		errorLog.Fatal(err)
	}

	app.wsManager = NewWebSocketManager()
	app.locationManager = NewLocationManager()
	app.locationHandler.Broadcast = app.locationManager.broadcast
	go app.wsManager.Run(db)
	go app.locationManager.Run()

	fs := http.FileServer(http.Dir("./uploads"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	allowedOrigins := map[string]struct{}{
		"http://localhost:3000": {},
		"http://localhost:3001": {},
		"http://localhost:5173": {},
		"http://localhost:5174": {},
	}

	c := cors.New(cors.Options{
		AllowOriginRequestFunc: func(r *http.Request, origin string) bool {
			if r.URL.Path == "/ws" || r.URL.Path == "/ws/location" ||
				r.URL.Path == "/ws/driver" || r.URL.Path == "/ws/passenger" {
				return true
			}
			_, ok := allowedOrigins[origin]
			return ok
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	})

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      addSecurityHeaders(c.Handler(app.routes())),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	if certFile != "" && keyFile != "" {
		infoLog.Printf("Starting TLS server on %s", *addr)
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
			errorLog.Fatal(err)
		}
		return
	}

	infoLog.Printf("Starting server on %s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		errorLog.Fatal(err)
	}
}
