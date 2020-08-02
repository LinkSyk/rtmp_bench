package util

import "errors"

var (
	FILEEOF     = errors.New("eof")
	EAGAIN      = errors.New("again")
	STREAMCLOSE = errors.New("stream close")
)
