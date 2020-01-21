package models

import "errors"

// Error объект для удобства обработки ошибок
type Error struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
}

// NewError создаёт новый объект ошибки
func NewError(code int, msg string) *Error {
	return &Error{
		Code:    code,
		Message: msg,
	}
}

const (
	// InternalDatabase неизвестная ошибка базы данных
	InternalDatabase = 700 + iota
	// RowNotFound запись в БД не найдена
	RowNotFound
	// ValidationFailed объект не прошел валидацию
	ValidationFailed
	// RowDuplication юзер с таким именем или почтой уже существует
	RowDuplication
	// ForeignKeyNotFound запись на которую ссылается не найдена
	ForeignKeyNotFound
	// ForeignKeyConflict запись на которую ссылаемся некорректна(например родитель в другом треде)
	ForeignKeyConflict
)

const (
	PgxOK            = ""
	PgxErrNotNull    = "23502"
	PgxErrForeignKey = "23503"
	PgxErrUnique     = "23505"
	NoRowsInResult   = "no rows in result set"
)

// Ошибки запросов
var (
	ForumIsExist			 = errors.New("Forum was created earlier")
	ForumNotFound			 = errors.New("Forum not found")
	ForumOrAuthorNotFound	 = errors.New("Forum or Author not found")
	UserNotFound			 = errors.New("User not found")
	UserIsExist				 = errors.New("User was created earlier")
	UserUpdateConflict		 = errors.New("User not updated")
	ThreadIsExist			 = errors.New("Thread was created earlier")
	ThreadNotFound			 = errors.New("Thread not found")
	PostParentNotFound		 = errors.New("No parent for thread")
	PostNotFound			 = errors.New("Post not found")
)
