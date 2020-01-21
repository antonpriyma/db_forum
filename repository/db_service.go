package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
)

type DBService struct {
	DB *pgx.ConnPool
}

func NewDBService() *DBService {
	return &DBService{}
}
func(s *DBService) GetStatus() (*models.Status, *models.Error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "can not open status tx")
	}
	defer tx.Rollback()

	// можно сделать на рефлексии, но зачем так тормозить
	status := &models.Status{}
	row := tx.QueryRow(`SELECT count(*) FROM forums`)
	if err = row.Scan(&status.Forum); err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	row = tx.QueryRow(`SELECT count(*) FROM posts`)
	if err = row.Scan(&status.Post); err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	row = tx.QueryRow(`SELECT count(*) FROM threads`)
	if err = row.Scan(&status.Thread); err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	row = tx.QueryRow(`SELECT count(*) FROM users`)
	if err = row.Scan(&status.User); err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	return status, nil
}

func(s *DBService) Load() *models.Error {
	_, err := s.DB.Exec(`
TRUNCATE users, forums, threads, posts, votes, forum_users;
`)
	if err != nil {
		return models.NewError(models.InternalDatabase, err.Error())
	}

	return nil
}

func ErrorCode(err error) (string) {
	pgerr, ok := err.(pgx.PgError)
	if !ok {
		return models.PgxOK
	}
	return pgerr.Code
}