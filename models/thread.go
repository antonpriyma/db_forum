package models

import (
	"log"
	"regexp"
	"time"
)

//easyjson:json
type Thread struct {
	ID      int64      `json:"id"`
	Slug    *string    `json:"slug"`
	Title   string     `json:"title"`
	Message string     `json:"message"`
	Votes   int32      `json:"votes"`
	Created *time.Time `json:"created"`
	Author  string     `json:"author"`
	Forum   string     `json:"forum"`
}

//easyjson:json
type Threads []*Thread

//easyjson:json
type Vote struct {
	Nickname  string `json:"nickname"`
	Voice     int    `json:"voice"`
	VoiceImpl bool
}

var (
	threadSlugRegexp *regexp.Regexp
)

func init() {
	var err error
	threadSlugRegexp, err = regexp.Compile(`^(\d|\w|-|_)*(\w|-|_)(\d|\w|-|_)*$`)
	if err != nil {
		log.Fatalf("slug regexp err: %s", err.Error())
	}
}

// Validate проверка полей
func (t *Thread) Validate() *Error {
	if !((t.Slug == nil || (t.Slug != nil && threadSlugRegexp.MatchString(*t.Slug))) &&
		t.Title != "" && t.Message != "") {
		return NewError(ValidationFailed, "validation failed")
	}

	return nil
}

