package index

import "errors"

var ErrProfileExists = errors.New("profile already exists")
var ErrProfileNotFound = errors.New("profile does not exist")
