package handlers

import (
	"strconv"
	"sync"
	"time"

	"github.com/p1xelse/goredis/internal/resp"
)

var Handlers = map[string]func([]resp.Value) resp.Value{
	"PING": ping,
	"ECHO": echo,
	"SET":  set,
	"GET":  get,
}

// TODO вынести хранилище в юзкейс чтобы там все обрабатывалось.

func ping(args []resp.Value) resp.Value {
	if len(args) > 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'ping' command"}
	}
	if len(args) == 0 {
		return resp.Value{Typ: resp.ValueTypeString, Str: "PONG"}
	}

	return resp.Value{Typ: resp.ValueTypeString, Str: args[0].Bulk}
}

func echo(args []resp.Value) resp.Value {
	if len(args) == 0 || len(args) > 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'echo' command"}
	}

	return resp.Value{Typ: resp.ValueTypeString, Str: args[0].Bulk}
}

type DataValue struct {
	value string
}

var data = map[string]string{}
var dataMu sync.RWMutex

var expirationKeys = map[string]int64{}
var expirationKeysMu sync.RWMutex

func set(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'set' command"}
	}

	var expirationTime int64
	for i := 2; i < len(args); i++ {
		if args[i].Bulk == "EX" || args[i].Bulk == "PX" {
			// TODO error count args.
			n, err := strconv.ParseInt(args[i+1].Bulk, 10, 64)
			if err != nil {
				return resp.Value{Typ: resp.ValueTypeError, Str: "wrong value for argument 'EX'"}
			}

			ttl := time.Duration(n) * time.Second
			if args[i].Bulk == "PX" {
				ttl = time.Duration(n) * time.Millisecond
			}

			expirationTime = time.Now().Add(ttl).UnixMilli()
		}
	}

	dataMu.Lock()
	data[args[0].Bulk] = args[1].Bulk
	dataMu.Unlock()

	if expirationTime > 0 {
		expirationKeysMu.Lock()
		expirationKeys[args[0].Bulk] = expirationTime
		expirationKeysMu.Unlock()
	}

	return resp.Value{Typ: resp.ValueTypeString, Str: "OK"}
}

func get(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'get' command"}
	}

	dataMu.RLock()
	defer dataMu.RUnlock()
	v, ok := data[args[0].Bulk]
	if !ok {
		return resp.Value{Typ: resp.ValueTypeNull}
	}

	return resp.Value{Typ: resp.ValueTypeBulkString, Bulk: v}
}

func ExpireJob() {
	for {
		for k, ex := range expirationKeys {
			if time.Now().UnixMilli() >= ex {
				dataMu.Lock()
				delete(expirationKeys, k)
				dataMu.Unlock()

				expirationKeysMu.Lock()
				delete(data, k)
				expirationKeysMu.Unlock()
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}
