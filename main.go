package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgxpool"
)

type handler struct {
	pool *pgxpool.Pool
}

// some long operation or some external call
func someLongMathOperationOrSomeLongExternalCall() {
	time.Sleep(time.Second)
}

// here context has come from outside and means that
// we have to finish our operations in context deadline
// because client wont wait any longer
// and no operation after deadline makes sense
func (h *handler) someHttpOrGrpcHandler(ctx context.Context) error {
	rows, err := h.pool.Query(ctx, "select some_column from some_table")
	// no error here
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var columns []int64
	for rows.Next() {
		var i int64
		if err := rows.Scan(&i); err != nil {
			return err
		}
		columns = append(columns, i)
	}

	if err = rows.Err(); err != nil {
		return err
	}

	// if context is done here, error from pgx is not returned
	// so we have to check context manually
	if err := ctx.Err(); err != nil {
		return err
	}

	// we have to check context before this function
	// because if we wont check it and context is already canceled
	// we would just waste our resources for nothing
	// because client doesnt want our response already
	someLongMathOperationOrSomeLongExternalCall()

	return nil
}

func NewHandler() *handler {
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

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	return &handler{pool: pool}
}
