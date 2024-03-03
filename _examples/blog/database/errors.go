package database

import "errors"

var (
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrSessionNotExist      = errors.New("session does not exist")
	ErrBlogNotExist         = errors.New("blog does not exist")
	ErrUserNotExist         = errors.New("user does not exist")
)
