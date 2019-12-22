package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/mailru/easyjson"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type HandlersUtils struct {
	log *logrus.Logger
}

type DBConfig struct {
	DBName     string
	DBUser     string
	DBPassword string
	Server     string
}


func(u *HandlersUtils) DecodeEasyjson(body io.Reader, v easyjson.Unmarshaler) error {
	return easyjson.UnmarshalFromReader(body, v)
}

// WriteEasyjson принимает структуру для easyjson, формирует и отправляет json ответ,
// если перед отправкой что-то ломается, отправляет 500
func(u *HandlersUtils) WriteEasyjson(w http.ResponseWriter, code int, v easyjson.Marshaler) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_, err := easyjson.MarshalToWriter(v, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func ConnectDatabase(config DBConfig) (*sql.DB, error) {
	dbinfo := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Server, config.DBUser, config.DBPassword, config.DBName)

	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		return db, err
	}
	if db == nil {
		return db, errors.New("Can not connect to database")
	}
	return db, nil
}


