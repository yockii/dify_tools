package util

import "github.com/rs/xid"

func NewShortID() string {
	return xid.New().String()
}
