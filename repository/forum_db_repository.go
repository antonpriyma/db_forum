package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
)

type ForumRepository interface {
	Create(forum *models.Forum) (*models.Forum, *models.Error)
	GetForumBySlug(slug string) (*models.Forum, *models.Error)
}

type ForumRepositoryImpl struct{
	db *pgx.ConnPool
}

func NewForumRepositoryImpl(db *pgx.ConnPool) ForumRepository {
	return &ForumRepositoryImpl{db: db}
}

func (r *ForumRepositoryImpl) Create(forum *models.Forum) (*models.Forum, *models.Error) {
	if validateError := forum.Validate(); validateError != nil {
		return nil, validateError
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "can not open 'forum create' transaction")
	}
	defer tx.Rollback()

	usedSlug, _ := getForumBySlugImpl(tx, forum.Slug)
	if usedSlug != nil {
		return usedSlug, models.NewError(models.RowDuplication, "slug is already used!")
	}

	// слегка странное решение, чтобы обеспечить сопадение nickname в таблице users и forums,
	// а также сохранить регистронезависимость;
	newRow, err := tx.Query(`INSERT INTO forums (slug, title, owner) VALUES ($1, $2, (SELECT nickname FROM users WHERE nickname = $3)) RETURNING id, owner`,
		forum.Slug, forum.Title, forum.Owner)
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	if !newRow.Next() {
		if pgerr, ok := newRow.Err().(pgx.PgError); ok && pgerr.Code == "23502" {
			return nil, models.NewError(models.ForeignKeyNotFound, pgerr.Error())
		}

		return nil, models.NewError(models.InternalDatabase, newRow.Err().Error())
	}
	// обновляем структуру, чтобы она содержала валидное имя создателя(учитывая регистр)
	// и валидный ID
	if err = newRow.Scan(&forum.ID, &forum.Owner); err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	newRow.Close()

	if err = tx.Commit(); err != nil {
		return nil, models.NewError(models.InternalDatabase, "forum create transaction commit error")
	}

	return nil, nil
}

func getForumBySlugImpl(q Queryer, slug string) (*models.Forum, *models.Error) {
	forum := &models.Forum{}
	row := q.QueryRow(`SELECT f.id, f.slug, f.title, f.posts, f.threads, f.owner FROM forums f WHERE slug = $1`, slug)
	if err := row.Scan(&forum.ID, &forum.Slug, &forum.Title,
		&forum.Posts, &forum.Threads, &forum.Owner); err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.NewError(models.RowNotFound, "row does not found")
		}

		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	return forum, nil
}


func (r *ForumRepositoryImpl) GetForumBySlug(slug string) (*models.Forum, *models.Error) {
	return getForumBySlugImpl(r.db, slug)
}

