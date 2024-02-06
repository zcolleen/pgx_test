package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	cfg, err := pgxpool.ParseConfig("postgres://dbuser:dbpassword@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		panic(err)
	}

	cfg.ConnConfig.BuildContextWatcherHandler = func(conn *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.DeadlineContextWatcherHandler{
			Conn: conn.Conn(),
			// long enough delay to let query finish
			DeadlineDelay: time.Second,
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
	defer cancel()

	rows, err := pool.Query(ctx, "select pg_sleep(0.2)")
	// no error here
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
	}

	// still no error while it should be because context is already canceled
	// in unchanged Handler there will be context deadline exceeded error
	if err = rows.Err(); err != nil {
		panic(err)
	}
	log.Println("finished without errors")
}
