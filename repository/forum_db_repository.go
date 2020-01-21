package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
	"log"
)

const (
	createForumSQL = `
		INSERT INTO forums (slug, title, "user")
		VALUES ($1, $2, (
			SELECT nickname FROM users WHERE nickname = $3
		)) 
		RETURNING "user"
	`

	getForumSQL = `
		SELECT slug, title, "user", posts, threads
		FROM forums
		WHERE slug = $1
	`

	createForumThreadSQL = `
		INSERT INTO threads (author, created, message, title, slug, forum)
		VALUES ($1, $2, $3, $4, $5, (SELECT slug FROM forums WHERE slug = $6)) 
		RETURNING author, created, forum, id, message, title
	`

	getForumThreadsSinceSQL = `
		SELECT author, created, forum, id, message, slug, title, votes
		FROM threads
		WHERE forum = $1 AND created >= $2::TEXT::TIMESTAMPTZ
		ORDER BY created
		LIMIT $3::TEXT::INTEGER
	`
	getForumThreadsDescSinceSQL = `
		SELECT author, created, forum, id, message, slug, title, votes
		FROM threads
		WHERE forum = $1 AND created <= $2::TEXT::TIMESTAMPTZ
		ORDER BY created DESC
		LIMIT $3::TEXT::INTEGER
	`
	getForumThreadsSQL = `
		SELECT author, created, forum, id, message, slug, title, votes
		FROM threads
		WHERE forum = $1
		ORDER BY created
		LIMIT $2::TEXT::INTEGER
	`
	getForumThreadsDescSQL = `
		SELECT author, created, forum, id, message, slug, title, votes
		FROM threads
		WHERE forum = $1
		ORDER BY created DESC
		LIMIT $2::TEXT::INTEGER
	`
	getForumUsersSinceSQl = `
		SELECT forum_user, fullname, about, email
		FROM forum_users
		WHERE forum = $1
		AND LOWER(forum_user) > LOWER($2::TEXT)
		ORDER BY forum_user
		LIMIT $3::TEXT::INTEGER
	`
	getForumUsersDescSinceSQl = `
		SELECT forum_user, fullname, about, email
		FROM forum_users
		WHERE forum = $1
		AND LOWER(forum_user) < LOWER($2::TEXT)
		ORDER BY forum_user DESC
		LIMIT $3::TEXT::INTEGER
	`
	getForumUsersSQl = `
		SELECT forum_user, fullname, about, email
		FROM forum_users
		WHERE forum = $1
		ORDER BY forum_user
		LIMIT $2::TEXT::INTEGER
	`
	getForumUsersDescSQl = `
		SELECT forum_user, fullname, about, email
		FROM forum_users
		WHERE forum = $1
		ORDER BY forum_user DESC
		LIMIT $2::TEXT::INTEGER
	`
)

type ForumRepository interface {
	Create(forum *models.Forum) (*models.Forum, error)
	GetForumBySlug(slug string) (*models.Forum, error)
	GetForumUsersDB(slug, limit, since, desc string) (*models.Users, error)
}

type ForumRepositoryImpl struct{
	db *pgx.ConnPool
}

var QueryForumWithSince = map[string]string{
	"true":  getForumThreadsDescSinceSQL,
	"false": getForumThreadsSinceSQL,
}

var QueryForumNoSince = map[string]string{
	"true":  getForumThreadsDescSQL,
	"false": getForumThreadsSQL,
}

var queryForumUserWithSince = map[string]string{
	"true":  getForumUsersDescSinceSQl,
	"false": getForumUsersSinceSQl,
}

var queryForumUserNoSince = map[string]string{
	"true":  getForumUsersDescSQl,
	"false": getForumUsersSQl,
}

func (r *ForumRepositoryImpl) GetForumUsersDB(slug, limit, since, desc string) (*models.Users, error) {
	var rows *pgx.Rows
	var err error

	if since != "" {
		query := queryForumUserWithSince[desc]
		rows, err = r.db.Query(query, slug, since, limit)
	} else {
		query := queryForumUserNoSince[desc]
		rows, err = r.db.Query(query, slug, limit)
	}
	defer rows.Close()

	if err != nil {
		log.Println(err)
		return nil, models.ForumNotFound
	}

	users := models.Users{}
	for rows.Next() {
		u := models.User{}
		err = rows.Scan(
			&u.Nickname,
			&u.Fullname,
			&u.About,
			&u.Email,
		)
		users = append(users, &u)
	}

	if len(users) == 0 {
		_, err := r.GetForumBySlug(slug)
		if err != nil {
			return nil, models.ForumNotFound
		}
	}
	return &users, nil
}

func NewForumRepositoryImpl(db *pgx.ConnPool) ForumRepository {
	return &ForumRepositoryImpl{db: db}
}

func (r *ForumRepositoryImpl) Create(forum *models.Forum) (*models.Forum, error) {
	err := r.db.QueryRow(
		createForumSQL,
		&forum.Slug,
		&forum.Title,
		&forum.Owner,
	).Scan(&forum.Owner)

	switch ErrorCode(err) {
	case models.PgxOK:
		return forum, nil
	case models.PgxErrUnique:
		forum, _ := r.GetForumBySlug(forum.Slug)
		return forum, models.ForumIsExist
	case models.PgxErrNotNull:
		return nil, models.UserNotFound
	default:
		return nil, err
	}
}




func (r *ForumRepositoryImpl) GetForumBySlug(slug string) (*models.Forum, error) {
	f := models.Forum{}

	err := r.db.QueryRow(
		getForumSQL,
		slug,
	).Scan(
		&f.Slug,
		&f.Title,
		&f.Owner,
		&f.Posts,
		&f.Threads,
	)

	if err != nil {
		return nil, models.ForumNotFound
	}

	return &f, nil
}

