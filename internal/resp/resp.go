package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

type ValueType int

const (
	ValueTypeString ValueType = iota
	ValueTypeInteger
	ValueTypeBulkString
	ValueTypeArray
	ValueTypeError
	ValueTypeNull
)

const (
	STRING  = '+'
	ERROR   = '-'
	INTEGER = ':'
	BULK    = '$'
	ARRAY   = '*'
)

type Value struct {
	Typ   ValueType
	Str   string
	Num   int
	Bulk  string
	Array []Value
}

func (v Value) Print() {
	fmt.Println("type:", v.Typ)
	fmt.Println("Str:", v.Str)
	fmt.Println("Num:", v.Num)
	fmt.Println("Bulk:", v.Bulk)
	fmt.Println("Array:")
	for _, v := range v.Array {
		v.Print()
	}
}

type Resp struct {
	reader *bufio.Reader
	writer io.Writer
}

func NewResp(rd io.Reader, wr io.Writer) *Resp {
	return &Resp{
		reader: bufio.NewReader(rd),
		writer: wr,
	}
}

func (r *Resp) Read() (Value, error) {
	_type, err := r.reader.ReadByte()

	if err != nil {
		return Value{}, err
	}

	switch _type {
	case ARRAY:
		return r.readArray()
	case BULK:
		return r.readBulk()
	default:
		fmt.Printf("Unknown type: %v", string(_type))
		return Value{}, nil
	}
}

func (r *Resp) Write(v Value) error {
	var bytes = v.Marshal()

	_, err := r.writer.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

func (v Value) Marshal() []byte {
	switch v.Typ {
	case ValueTypeArray:
		return v.marshalArray()
	case ValueTypeBulkString:
		return v.marshalBulk()
	case ValueTypeString:
		return v.marshalString()
	case ValueTypeNull:
		return v.marshallNull()
	case ValueTypeError:
		return v.marshallError()
	default:
		return []byte{}
	}
}

func (v Value) marshalString() []byte {
	var bytes []byte
	bytes = append(bytes, STRING)
	bytes = append(bytes, '"')
	bytes = append(bytes, v.Str...)
	bytes = append(bytes, '"')
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (v Value) marshalBulk() []byte {
	var bytes []byte
	bytes = append(bytes, BULK)
	bytes = append(bytes, strconv.FormatInt(int64(len(v.Bulk)), 10)...)
	bytes = append(bytes, '\r', '\n')
	bytes = append(bytes, v.Bulk...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (v Value) marshalArray() []byte {
	var bytes []byte
	bytes = append(bytes, ARRAY)
	bytes = append(bytes, strconv.FormatInt(int64(len(v.Array)), 10)...)
	bytes = append(bytes, '\r', '\n')
	for _, vv := range v.Array {
		bytes = append(bytes, vv.Marshal()...)
	}

	return bytes
}

func (v Value) marshallError() []byte {
	var bytes []byte
	bytes = append(bytes, ERROR)
	bytes = append(bytes, v.Str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (v Value) marshallNull() []byte {
	return []byte("$-1\r\n")
}

func (r *Resp) readLine() (line []byte, n int, err error) {
	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			return nil, 0, err
		}
		n += 1
		line = append(line, b)
		if len(line) >= 2 && line[len(line)-2] == '\r' {
			break
		}
	}
	return line[:len(line)-2], n, nil
}

func (r *Resp) readInteger() (int64, int, error) {
	numStr, n, err := r.readLine()
	if err != nil {
		return 0, 0, err
	}

	num, err := strconv.ParseInt(string(numStr), 10, 64)
	if err != nil {
		return 0, n, err
	}

	return num, n, err
}

func (r *Resp) readArray() (Value, error) {
	v := Value{}
	v.Typ = ValueTypeArray

	length, _, err := r.readInteger()
	if err != nil {
		return v, err
	}

	v.Array = make([]Value, length)
	for i := int64(0); i < length; i++ {
		val, err := r.Read()
		if err != nil {
			return v, err
		}

		v.Array[i] = val
	}

	return v, nil
}

func (r *Resp) readBulk() (Value, error) {
	v := Value{}
	v.Typ = ValueTypeBulkString
	length, _, err := r.readInteger()
	if err != nil {
		return v, err
	}

	bulk := make([]byte, length)

	_, err = r.reader.Read(bulk)
	if err != nil {
		return Value{}, err
	}

	v.Bulk = string(bulk)

	// Read the trailing CRLF
	_, _, err = r.readLine()
	if err != nil {
		return Value{}, err
	}

	return v, nil
}
