package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"gophermart/internal/auth"
	"gophermart/internal/config"
	"gophermart/internal/hash"
	"gophermart/internal/http"
	"gophermart/internal/server"
	"gophermart/internal/service"
	"gophermart/internal/storage"
	"log"
	"time"
)

func main() {
	var cfg config.Config
	err := cfg.Parse()
	if err != nil {
		log.Fatal(err)
	}

	db, err := newInPSQL("postgres://postgres:root@localhost:5432/gophermart?sslmode=disable")
	//db, err := newInPSQL(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}

	storages := storage.NewStorages(db)

	hasher := hash.NewSHA1Hasher(cfg.PasswordSalt)
	tokenManager, err := auth.NewManager(cfg.PasswordSalt)
	if err != nil {
		log.Fatal(err)
	}

	deps := service.Deps{
		Storages: storages,
		Hasher: hasher,
		TokenManager: tokenManager,
		AccessTokenTTL: 365 * 24 * time.Hour,
		AccrualAddress: cfg.AccrualAddress,
	}

	services := service.NewServices(deps)

	//TODO remove
	//err = services.Accrual.(*service.AccrualService).FillWithTestData(nil)
	//if err != nil {
	//	log.Fatal(err)
	//}

	handlers := http.NewHandler(services, tokenManager)

	// HTTP Server
	server := server.NewServer(&cfg, handlers.Init()) //TODO
	//server := server.NewServer(&cfg, handlers.Init(&cfg)) //TODO
	server.Run()
}

func newInPSQL(databaseDSN string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseDSN)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err = createTable(db); err != nil {
		log.Fatal(err)
	}
	return db, nil
}

func createTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id serial primary key,
		login text not null unique,
		password text not null,
        "current" float not null default 0,
        withdrawn float not null  default 0
    );
	CREATE TABLE IF NOT EXISTS orders (
		"number" text primary key unique,
		user_id int not null references users(id),
		status text not null,
		accrual float,
		uploaded_at timestamp
	);
	CREATE TABLE IF NOT EXISTS withdrawals (
		user_id int not null references users(id),
		"order_number" text not null unique,
		"sum" float not null,
		processed_at timestamp
	);`
	_, err := db.Exec(query)
	return err
}

