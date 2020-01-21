package models

import (
	"time"
)

//easyjson:json
type Post struct {

	Author string `json:"author"`
	Created time.Time `json:"created,omitempty"`
	Forum string `json:"forum,omitempty"`
	ID int64 `json:"id,omitempty"`
	IsEdited bool `json:"isEdited,omitempty"`
	Message string `json:"message"`
	Parent int64 `json:"parent,omitempty"`
	Thread int32 `json:"thread,omitempty"`
}

type PostUpdate struct {
	Message string `json:"message,omitempty"`
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

