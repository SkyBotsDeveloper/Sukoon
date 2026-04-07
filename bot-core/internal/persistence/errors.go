package persistence

import "errors"

var ErrCloneLimitReached = errors.New("only one clone per owner is allowed")
