package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkerPool struct {
	workerNum int
	tasks     chan func()
}

func NewWorkerPool(ctx context.Context, workerNum int) *WorkerPool {
	w := &WorkerPool{
		tasks: make(chan func(), workerNum),
	}
	for i := 0; i < workerNum; i++ {
		go func() {
			for {
				select {
				case t := <-w.tasks:
					t()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return w
}

func (w *WorkerPool) Exec(f func()) {
	w.tasks <- f
}

func execTx(ctx context.Context, pool *pgxpool.Pool) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if errTx := tx.Rollback(ctx); errTx != nil {
			if !errors.Is(errTx, pgx.ErrTxClosed) {
				err = errTx
			}
		}
	}()

	_, err = tx.Exec(ctx, "select pg_sleep(0.5)")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	cancel()

	return tx.Commit(ctx)
}

func exec(ctx context.Context, pool *pgxpool.Pool) error {
	r, err := pool.Query(ctx, "select * from pgx_table limit 10")
	if err != nil {
		return err
	}
	var id, age int
	var name, addr string
	for r.Next() {
		if err := r.Scan(&id, &name, &age, &addr); err != nil {
			return err
		}
	}

	return r.Err()
}

func metrics(a *atomic.Uint64, failed *atomic.Uint64, deadlineFailures *atomic.Uint64) {
	http.HandleFunc("/total", func(writer http.ResponseWriter, request *http.Request) {
		res := strconv.FormatUint(a.Load(), 10)

		writer.Write([]byte(res))
	})

	http.HandleFunc("/failed", func(writer http.ResponseWriter, request *http.Request) {
		res := strconv.FormatUint(failed.Load(), 10)

		writer.Write([]byte(res))
	})

	http.HandleFunc("/rps", func(writer http.ResponseWriter, request *http.Request) {
		val1 := a.Load()
		time.Sleep(time.Second)
		val2 := a.Load()

		res := strconv.FormatUint(val2-val1, 10)
		writer.Write([]byte(res))
	})

	http.HandleFunc("/failed/deadline", func(writer http.ResponseWriter, request *http.Request) {
		total := a.Load()
		dFailed := deadlineFailures.Load()

		persentage := float64(dFailed) / float64(total)
		res := strconv.FormatFloat(persentage, 'g', 10, 64)

		writer.Write([]byte(res))
	})

	go func() {
		panic(http.ListenAndServe("0.0.0.0:8081", nil))
	}()
}

func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://dbuser:dbpassword@192.168.1.3:5432/postgres?sslmode=disable&default_query_exec_mode=simple_protocol&no_closing_conn_mode=true&conn_cleanup_timeout=200s")
	if err != nil {
		panic(err)
	}

	total := &atomic.Uint64{}
	deadlineFailures := &atomic.Uint64{}
	failed := &atomic.Uint64{}
	metrics(total, failed, deadlineFailures)

	w := NewWorkerPool(ctx, 1)

	f := func() {
		defer total.Add(1)

		c, cancel := context.WithTimeout(ctx, time.Second*1)
		defer cancel()

		err = exec(c, pool)
		if err != nil {

			failed.Add(1)
			if errors.Is(err, context.DeadlineExceeded) {
				deadlineFailures.Add(1)
			} else {
				log.Println(err)
			}

			return
		}

		total.Add(1)
	}
	for {
		w.Exec(f)
	}
}
