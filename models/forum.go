package models

import (
	"log"
	"regexp"
)

// Forum информация о форуме.
//easyjson:json
type Forum struct {
	ID      int64  `json:"-"`
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Posts   int64  `json:"posts"`
	Threads int32  `json:"threads"`
	Owner   string `json:"user"`
}

var (
	forumSlugRegexp *regexp.Regexp
)

func init() {
	var err error
	forumSlugRegexp, err = regexp.Compile(`^(\d|\w|-|_)*(\w|-|_)(\d|\w|-|_)*$`)
	if err != nil {
		log.Fatalf("slug regexp err: %s", err.Error())
	}
}

// Validate проверка полей
func (f *Forum) Validate() *Error {
	if !(forumSlugRegexp.MatchString(f.Slug) &&
		f.Title != "") {
		return NewError(ValidationFailed, "validation failed")
	}

	return nil
}

