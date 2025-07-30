package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/p1xelse/goredis/internal/handlers"
	respPkg "github.com/p1xelse/goredis/internal/resp"
)

func main() {
	ln, err := net.Listen("tcp", ":6379") // 6379 — стандартный порт Redis
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	fmt.Println("Listening on :6379")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		// Обрабатываем каждое соединение в отдельной горутине
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		resp := respPkg.NewResp(conn, conn)
		value, err := resp.Read()
		if err != nil {
			fmt.Println(err)
			return
		}

		value.Print()

		if value.Typ != respPkg.ValueTypeArray {
			fmt.Println("Invalid request, expected array")
			continue
		}

		if len(value.Array) == 0 {
			fmt.Println("Invalid request, expected array length > 0")
			continue
		}

		cmd := strings.ToUpper(value.Array[0].Bulk)
		handler, ok := handlers.Handlers[cmd]
		if !ok {
			resp.Write(respPkg.Value{Typ: respPkg.ValueTypeError, Str: fmt.Sprintf("ERR unknown command '%s'", value.Array[0].Bulk)})
			continue
		}

		resp.Write(handler(value.Array[1:]))
	}
}
