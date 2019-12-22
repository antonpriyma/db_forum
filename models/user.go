package models

import (
	"log"
	"regexp"
)

// User информация о пользователе
//easyjson:json
type User struct {
	ID       int64  `json:"-"`
	Nickname string `json:"nickname"`
	Fullname string `json:"fullname"`
	About    string `json:"about"`
	Email    string `json:"email"`
}

// UpdateUserFields структура для обновления полей юзера
//easyjson:json
type UpdateUserFields struct {
	Fullname *string `json:"fullname"`
	About    *string `json:"about"`
	Email    *string `json:"email"`
}

// Users несколько юзеров
//easyjson:json
type Users []*User

var (
	nicknameRegexp *regexp.Regexp
	emailRegexp    *regexp.Regexp
)

func init() {
	var err error
	nicknameRegexp, err = regexp.Compile(`^[a-zA-Z0-9_.]+$`)
	if err != nil {
		log.Fatalf("nickname regexp err: %s", err.Error())
	}

	emailRegexp, err = regexp.Compile("^[a-zA-Z0-9.!#$%&''*+/=?^_`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
	if err != nil {
		log.Fatalf("email regexp err: %s", err.Error())
	}
}

// Validate проверка полей
func (u *User) Validate() *Error {
	if !(nicknameRegexp.MatchString(u.Nickname) &&
		emailRegexp.MatchString(u.Email) &&
		u.Fullname != "") {
		return NewError(ValidationFailed, "validation failed")
	}

	return nil
}

