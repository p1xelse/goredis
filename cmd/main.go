package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	errGroup "golang.org/x/sync/errgroup"

	"github.com/p1xelse/goredis/internal/handlers"
	respPkg "github.com/p1xelse/goredis/internal/resp"
	"github.com/p1xelse/goredis/internal/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

		select {
		case <-quit:
			log.Println("shutting down...")
			cancel()
		}
	}()

	ln, err := net.Listen("tcp", ":6379") // 6379 — стандартный порт Redis
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	fmt.Println("Listening on :6379")

	// Storage.
	st, err := storage.New(true)
	if err != nil {
		log.Fatal("creating storage:", err)
	}

	// Handler.
	h := handlers.New(st)

	eg := errGroup.Group{}

	eg.Go(func() error {
		return st.ExpireJob(ctx)
	})

	eg.Go(func() error {
		return serve(ctx, ln, h)
	})

	if err := eg.Wait(); err != nil {
		log.Fatal(err)
	}

	return
}

func serve(ctx context.Context, ln net.Listener, handler *handlers.Handler) (err error) {
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	eg := errGroup.Group{}
	defer func() {
		log.Println("waiting for close connections...")
		err = eg.Wait()
		return
	}()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("exiting for listen loop: %v", ctx.Err())
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("listener closed: %v", ctx.Err())
			}

			log.Println("accept error:", err)
			continue
		}
		// Обрабатываем каждое соединение в отдельной горутине
		eg.Go(func() error {
			handleConnection(ctx, conn, handler)
			return nil
		})
	}
}

func handleConnection(ctx context.Context, conn net.Conn, handler *handlers.Handler) {
	var closeOnce sync.Once
	safeClose := func() {
		closeOnce.Do(func() {
			conn.Close()
		})
	}

	defer safeClose()
	// Закроем при отмене контекста.
	go func() {
		<-ctx.Done()
		safeClose()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp := respPkg.NewResp(conn, conn)
		value, err := resp.Read()
		if err != nil {
			log.Printf("resp read error: %v\n", err)
			return
		}

		if value.Typ != respPkg.ValueTypeArray {
			fmt.Println("Invalid request, expected array")
			continue
		}

		if len(value.Array) == 0 {
			fmt.Println("Invalid request, expected array length > 0")
			continue
		}

		cmd := strings.ToUpper(value.Array[0].Bulk)
		resp.Write(handler.Handle(cmd, value.Array[1:]))
	}
}
