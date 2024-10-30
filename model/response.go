package model

import (
	"net/http"
)

type Success struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data"`
}

type Error struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func NewSuccess(data interface{}) Success {
	return Success{
		Code: http.StatusOK,
		Data: data,
	}
}

func NewError(msg string, code ...int) Error {
	var c = 500
	if len(code) > 0 {
		c = code[0]
	}
	return Error{
		Code: c,
		Msg:  msg,
	}
}

func NewErrorWithData(msg string, data interface{}, code ...int) Error {
	var c = 500
	if len(code) > 0 {
		c = code[0]
	}
	return Error{
		Code: c,
		Msg:  msg,
		Data: data,
	}
}
