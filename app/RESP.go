package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type DataType int

const (
	NEWLINE               = "\r\n"
	SimpleString DataType = iota
	Error
	Integer
	BulkString
	Array
)

func strip(s string) string {
	// no need to check if NEWLINE is suffix of s
	return s[:len(s)-len(NEWLINE)]
}

type RESP interface {
	datatype() DataType
	serialize() string
	response(data *SafeMap) string
}

type RESPNil struct{}

func (r *RESPNil) datatype() DataType {
	return BulkString
}

func (r *RESPNil) serialize() string {
	return fmt.Sprintf("$-1%s", NEWLINE)
}

func (r *RESPNil) response(data *SafeMap) string {
	return r.serialize()
}

type RESPSimpleString struct {
	data string
}

func parseSimpleString(reader *bufio.Reader) (*RESPSimpleString, error) {
	// + is already consumed
	s, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to parse simple string: " + err.Error())
	}
	return &RESPSimpleString{s}, nil
}

func (r *RESPSimpleString) datatype() DataType {
	return SimpleString
}

func (r *RESPSimpleString) serialize() string {
	return fmt.Sprintf("+%s%s", r.data, NEWLINE)
}

func (r *RESPSimpleString) response(data *SafeMap) string {
	return r.serialize()
}

type RESPError struct {
	eMsg string
}

func parseError(reader *bufio.Reader) (*RESPError, error) {
	// - is already consumed
	eMsg, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to parse error: " + err.Error())
	}
	return &RESPError{eMsg}, nil
}

func (r *RESPError) datatype() DataType {
	return Error
}

func (r *RESPError) serialize() string {
	return fmt.Sprintf("-%s%s", r.eMsg, NEWLINE)
}

func (r *RESPError) response(data *SafeMap) string {
	return r.serialize()
}

type RESPInteger struct {
	data int
}

func parseInteger(reader *bufio.Reader) (*RESPInteger, error) {
	// : is already consumed
	nStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to parse integer: " + err.Error())
	}
	i, err := strconv.Atoi(strip(nStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse integer: " + err.Error())
	}
	return &RESPInteger{i}, nil
}

func (r *RESPInteger) datatype() DataType {
	return Integer
}

func (r *RESPInteger) serialize() string {
	return fmt.Sprintf(":%d%s", r.data, NEWLINE)
}

func (r *RESPInteger) response(data *SafeMap) string {
	return r.serialize()
}

type RESPBulkString struct {
	length int
	data   string
}

func ParseBulkString(reader *bufio.Reader) (*RESPBulkString, error) {
	// $ is already consumed
	integer, err := parseInteger(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bulk string: " + err.Error())
	}
	n, result := integer.data, ""
	for len(result) < n {
		s, err := parseSimpleString(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bulk string: " + err.Error())
		}
		result = result + s.data
	}
	return &RESPBulkString{n, result[:n]}, nil
}

func (r *RESPBulkString) datatype() DataType {
	return BulkString
}

func (r *RESPBulkString) serialize() string {
	return fmt.Sprintf("$%d%s%s%s", r.length, NEWLINE, r.data, NEWLINE)
}

func (r *RESPBulkString) response(data *SafeMap) string {
	return r.serialize()
}

type RESPArray struct {
	length int
	data   []RESP
}

func parseArray(reader *bufio.Reader) (*RESPArray, error) {
	// * is already consumed
	integer, err := parseInteger(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bulk string: " + err.Error())
	}
	n, result := integer.data, []RESP{}
	for len(result) < n {
		r, err := ParseRESP(reader)
		if err != nil {
			return nil, fmt.Errorf("failed while parsing array element: " + err.Error())
		}
		result = append(result, r)
	}
	return &RESPArray{n, result}, nil
}

func (r *RESPArray) datatype() DataType {
	return Array
}

func (r *RESPArray) serialize() string {
	result := fmt.Sprintf("*%d%s", r.length, NEWLINE)
	for _, e := range r.data {
		result = result + e.serialize()
	}
	return result
}

func (r *RESPArray) response(data *SafeMap) string {
	switch r.data[0].datatype() {
	case BulkString:
		cmd, _ := r.data[0].(*RESPBulkString)
		switch strings.ToLower(cmd.data) {
		case "echo":
			return r.data[1].serialize()
		case "ping":
			return (&RESPBulkString{4, "PONG"}).serialize()
		case "set":
			key, ok := r.data[1].(*RESPBulkString)
			if !ok {
				return (&RESPError{"SET command expect bulk string for KEY"}).serialize()
			}
			value, ok := r.data[2].(*RESPBulkString)
			if !ok {
				return (&RESPError{"SET command expect bulk string for VALUE"}).serialize()
			}
			if len(r.data) == 5 {
				px, ok := r.data[3].(*RESPBulkString)
				if !ok || strings.ToLower(px.data) != "px" {
					return (&RESPError{"fail to parse PX parameter in SET command"}).serialize()
				}
				integer, ok := r.data[4].(*RESPBulkString)
				if !ok {
					return (&RESPError{"PX parameter in SET command expect bulk string"}).serialize()
				}
				n, err := strconv.Atoi(integer.data)
				if err != nil {
					return (&RESPError{"failed to parse expire milliseconds: " + integer.data}).serialize()
				}
				data.setWithExpiry(key.data, value.data, n)
			} else {
				data.set(key.data, value.data)
			}
			return (&RESPSimpleString{"OK"}).serialize()
		case "get":
			key, ok := r.data[1].(*RESPBulkString)
			if !ok {
				return (&RESPError{"SET command expect bulk string for KEY"}).serialize()
			}
			value, err := data.get(key.data)
			if err != nil {
				return (&RESPNil{}).serialize()
			}
			return (&RESPBulkString{len(value), value}).serialize()
		default:
			return (&RESPError{"Unsupported command: " + cmd.data}).serialize()
		}
	default:
		return r.serialize()
	}
}

func ParseRESP(reader *bufio.Reader) (RESP, error) {
	op, err := reader.ReadByte()
	if err != nil {
		if err.Error() == "EOF" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read op: " + err.Error())
	}
	switch string(op) {
	case "+":
		r, err := parseSimpleString(reader)
		if err != nil {
			return nil, err
		}
		return r, nil
	case "-":
		r, err := parseError(reader)
		if err != nil {
			return nil, err
		}
		return r, nil
	case ":":
		r, err := parseInteger(reader)
		if err != nil {
			return nil, err
		}
		return r, nil
	case "$":
		r, err := ParseBulkString(reader)
		if err != nil {
			return nil, err
		}
		return r, nil
	case "*":
		r, err := parseArray(reader)
		if err != nil {
			return nil, err
		}
		return r, nil
	default:
		return nil, fmt.Errorf("unexpected op character: " + string(op))
	}
}
