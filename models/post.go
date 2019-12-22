package models

import (
	"time"
)

//easyjson:json
type Post struct {
	ID       int64     `json:"id"`
	Message  string    `json:"message"`
	IsEdited bool      `json:"isEdited"`
	Author   string    `json:"author"`
	Forum    string    `json:"forum"`
	Thread   int64     `json:"thread"`
	Parent   int64     `json:"parent"`
	Created  time.Time `json:"created"`

	ParentImpl *int64
}

//easyjson:json
type Posts []*Post

//easyjson:json
type PostFull struct {
	Post   *Post   `json:"post"`
	Author *User   `json:"author"`
	Forum  *Forum  `json:"forum"`
	Thread *Thread `json:"thread"`
}

func (p *Post) Validate() *Error {
	if p.Message == "" {
		return NewError(ValidationFailed, "message is empty")
	}

	return nil
}

