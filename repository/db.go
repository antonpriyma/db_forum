package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
)

type Queryer interface {
	QueryRow(string, ...interface{}) *pgx.Row
	Query(string, ...interface{}) (*pgx.Rows, error)
}

type executer interface {
	Exec(string, ...interface{}) (pgx.CommandTag, error)
}

type exequeryer interface {
	Queryer
	executer
}

var db *pgx.ConnPool

func ConnetctDB(service *DBService, dbUser, dbPass, dbHost, dbName string) *models.Error {
	newDB, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     dbHost,
			User:     dbUser,
			Password: dbPass,
			Port:     5432,
			Database: dbName,
		},
		MaxConnections: 50,
	})
	if err != nil {
		return models.NewError(models.InternalDatabase, err.Error())
	}

	db = newDB
	service.DB = db
	if err := service.Load(); err != nil {
		return err
	}

	return nil
}

func GetDB() *pgx.ConnPool {
	return db
}
