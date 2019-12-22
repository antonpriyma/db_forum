package utils

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/mailru/easyjson"
)

// DecodeEasyjson принимает структуру для easyjson и тело запроса,
// парсит JSON в случае неудачи возвращает ошибку
func DecodeEasyjson(body io.Reader, v easyjson.Unmarshaler) error {
	return easyjson.UnmarshalFromReader(body, v)
}

// WriteEasyjson принимает структуру для easyjson, формирует и отправляет json ответ,
// если перед отправкой что-то ломается, отправляет 500
func WriteEasyjson(w http.ResponseWriter, code int, v json.Marshaler) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)


	body,err := json.Marshal(v)
	w.Write(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
