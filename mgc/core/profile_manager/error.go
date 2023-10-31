package profile_manager

import (
	"errors"
	"fmt"
)

var errorNameNotAllowed = fmt.Errorf("%s is not an allowed name", currentProfileNameFile)
var errorInvalidName = errors.New("name should only contain alphanumric characters, underscores or hypens")
var errorProfileAlreadyExists = errors.New("profile already exists")
var errorDeleteCurrentNotAllowed = errors.New("cannot delete current profile")
var errorCopyToSelf = errors.New("cannot copy to itself")
