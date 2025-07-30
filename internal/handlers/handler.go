package handlers

import (
	"github.com/p1xelse/goredis/internal/resp"
)

var Handlers = map[string]func([]resp.Value) resp.Value{
	"PING": ping,
	"ECHO": echo,
}

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
