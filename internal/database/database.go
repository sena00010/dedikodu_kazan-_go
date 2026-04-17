package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	_ "github.com/go-sql-driver/mysql"
)

func OpenMySQL(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db, db.PingContext(ctx)
}

func OpenRedis(addr, password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
}
