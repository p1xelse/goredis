package handlers

import (
	"strconv"
	"time"

	"github.com/p1xelse/goredis/internal/resp"
	storageDto "github.com/p1xelse/goredis/internal/storage"
	"github.com/p1xelse/goredis/internal/utils"
)

type storage interface {
	Set(key string, value string, opts storageDto.SetOptions) error
	Get(key string) (string, bool)
	Exists(keys []string) int
	Delete(keys []string) int
	Increment(key string) (int, error)
	Decrement(key string) (int, error)
	Save() error
}

type Handler struct {
	storage storage

	handlers map[string]func([]resp.Value) resp.Value
}

func New(storage storage) *Handler {
	h := &Handler{
		storage:  storage,
		handlers: make(map[string]func([]resp.Value) resp.Value),
	}

	h.handlers = map[string]func([]resp.Value) resp.Value{
		"PING":   h.ping,
		"ECHO":   h.echo,
		"SET":    h.set,
		"GET":    h.get,
		"EXISTS": h.exists,
		"DEL":    h.del,
		"INCR":   h.incr,
		"DECR":   h.decr,
		"SAVE":   h.save,
	}

	return h
}

func (h *Handler) Handle(cmd string, args []resp.Value) resp.Value {
	function, exist := h.handlers[cmd]
	if !exist {
		return resp.Value{Typ: resp.ValueTypeError, Str: "unknown command '" + cmd + "'"}
	}

	return function(args)
}

func (h *Handler) ping(args []resp.Value) resp.Value {
	if len(args) > 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'ping' command"}
	}
	if len(args) == 0 {
		return resp.Value{Typ: resp.ValueTypeString, Str: "PONG"}
	}

	return resp.Value{Typ: resp.ValueTypeString, Str: args[0].Bulk}
}

func (h *Handler) echo(args []resp.Value) resp.Value {
	if len(args) == 0 || len(args) > 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'echo' command"}
	}

	return resp.Value{Typ: resp.ValueTypeString, Str: args[0].Bulk}
}

func (h *Handler) set(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'set' command"}
	}

	var ttl *time.Duration
	var expirationTime *int64

	for i := 2; i < len(args); i++ {
		if args[i].Bulk == "EX" || args[i].Bulk == "PX" || args[i].Bulk == "EXAT" || args[i].Bulk == "PXAT" {
			// TODO error count args.
			n, err := strconv.ParseInt(args[i+1].Bulk, 10, 64)
			if err != nil {
				return resp.Value{Typ: resp.ValueTypeError, Str: "wrong value for argument 'EX'"}
			}

			ttl = utils.Ptr(time.Duration(n) * time.Second)
			if args[i].Bulk == "PX" {
				ttl = utils.Ptr(time.Duration(n) * time.Millisecond)
			}

			if args[i].Bulk == "EXAT" {
				expirationTime = utils.Ptr(time.UnixMilli(n * 1000).UnixMilli())
			}

			if args[i].Bulk == "PXAT" {
				expirationTime = utils.Ptr(time.UnixMilli(n).UnixMilli())
			}
		}
	}

	err := h.storage.Set(args[0].Bulk, args[1].Bulk, storageDto.SetOptions{
		TTL:        ttl,
		Expiration: expirationTime,
	})

	if err != nil {
		return resp.Value{Typ: resp.ValueTypeError, Str: err.Error()}
	}

	return resp.Value{Typ: resp.ValueTypeString, Str: "OK"}
}

func (h *Handler) get(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'get' command"}
	}

	v, ok := h.storage.Get(args[0].Bulk)
	if !ok {
		return resp.Value{Typ: resp.ValueTypeNull}
	}

	return resp.Value{Typ: resp.ValueTypeBulkString, Bulk: v}
}

func (h *Handler) exists(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'exists' command"}
	}
	keys := make([]string, 0, len(args))
	for _, k := range args {
		keys = append(keys, k.Bulk)
	}

	count := h.storage.Exists(keys)

	return resp.Value{Typ: resp.ValueTypeInteger, Num: count}
}

func (h *Handler) del(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'del' command"}
	}

	keys := make([]string, 0, len(args))
	for _, k := range args {
		keys = append(keys, k.Bulk)
	}

	count := h.storage.Delete(keys)

	return resp.Value{Typ: resp.ValueTypeInteger, Num: count}
}

func (h *Handler) incr(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'incr' command"}
	}

	incrValue, err := h.storage.Increment(args[0].Bulk)
	if err != nil {
		return resp.Value{Typ: resp.ValueTypeError, Str: err.Error()}
	}

	return resp.Value{Typ: resp.ValueTypeInteger, Num: incrValue}
}

func (h *Handler) decr(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'incr' command"}
	}

	decrValue, err := h.storage.Increment(args[0].Bulk)
	if err != nil {
		return resp.Value{Typ: resp.ValueTypeError, Str: err.Error()}
	}

	return resp.Value{Typ: resp.ValueTypeInteger, Num: int(decrValue)}
}

func (h *Handler) save(args []resp.Value) resp.Value {
	if len(args) > 0 {
		return resp.Value{Typ: resp.ValueTypeError, Str: "wrong number of arguments for 'save' command"}
	}

	if err := h.storage.Save(); err != nil {
		return resp.Value{Typ: resp.ValueTypeError, Str: err.Error()}
	}

	return resp.Value{Typ: resp.ValueTypeString, Str: "OK"}
}
