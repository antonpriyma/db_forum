package repository

import (
	"fmt"
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
	"strconv"
)

type ThreadDBRepository interface {
	Create(thread *models.Thread) (*models.Thread,error) // ok
	UpdateThreadDB(thread *models.ThreadUpdate, param string) (*models.Thread, error) //ok
	MakeThreadVoteDB(vote *models.Vote, param string) (*models.Thread, error) //ok
	GetThreadsByForum(slug, limit, since, desc string) (*models.Threads, error) //ok
	GetThread(param string) (*models.Thread, error) //ok
	parentExitsInOtherThread(parent int64, threadID int32) bool
	parentNotExists(parent int64) bool
}

func(p *ThreadDBRepositoryImpl) parentExitsInOtherThread(parent int64, threadID int32) bool {
	var t int64
	err := p.db.QueryRow(postID, parent, threadID).Scan(&t)

	if err != nil && err.Error() == models.NoRowsInResult {
		return false
	}
	return true
}

func(p *ThreadDBRepositoryImpl) parentNotExists(parent int64) bool {
	if parent == 0 {
		return false
	}

	var t int64
	err := p.db.QueryRow(`SELECT id FROM posts WHERE id = $1`, parent).Scan(&t)

	if err != nil {
		return true
	}
	return false
}

const (
	getThreadSlugSQL = `
		SELECT id, title, author, forum, message, votes, slug, created
		FROM threads
		WHERE slug = $1
	`
	getThreadIdSQL = `
		SELECT id, title, author, forum, message, votes, slug, created
		FROM threads
		WHERE id = $1
	`
	updateThreadSQL = `
		UPDATE threads
		SET title = coalesce(nullif($2, ''), title),
			message = coalesce(nullif($3, ''), message)
		WHERE slug = $1
		RETURNING id, title, author, forum, message, votes, slug, created
	`

	// getThreadPosts
	getPostsSienceDescLimitTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 AND (path < (SELECT path FROM posts WHERE id = $2::TEXT::INTEGER))
		ORDER BY path DESC
		LIMIT $3::TEXT::INTEGER
	`

	getPostsSienceDescLimitParentTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts p
		WHERE p.thread = $1 and p.path[1] IN (
			SELECT p2.path[1]
			FROM posts p2
			WHERE p2.thread = $1 AND p2.parent = 0 and p2.path[1] < (SELECT p3.path[1] from posts p3 where p3.id = $2)
			ORDER BY p2.path DESC
			LIMIT $3
		)
		ORDER BY p.path[1] DESC, p.path[2:]
	`

	getPostsSienceDescLimitFlatSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 AND id < $2::TEXT::INTEGER
		ORDER BY id DESC
		LIMIT $3::TEXT::INTEGER
	`

	getPostsSienceLimitTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 AND (path > (SELECT path FROM posts WHERE id = $2::TEXT::INTEGER))
		ORDER BY path
		LIMIT $3::TEXT::INTEGER
	`

	getPostsSienceLimitParentTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts p
		WHERE p.thread = $1 and p.path[1] IN (
			SELECT p2.path[1]
			FROM posts p2
			WHERE p2.thread = $1 AND p2.parent = 0 and p2.path[1] > (SELECT p3.path[1] from posts p3 where p3.id = $2::TEXT::INTEGER)
			ORDER BY p2.path
			LIMIT $3::TEXT::INTEGER
		)
		ORDER BY p.path
	`
	getPostsSienceLimitFlatSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 AND id > $2::TEXT::INTEGER
		ORDER BY id
		LIMIT $3::TEXT::INTEGER
	`
	// without sience
	getPostsDescLimitTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 
		ORDER BY path DESC
		LIMIT $2::TEXT::INTEGER
	`
	getPostsDescLimitParentTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 AND path[1] IN (
			SELECT path[1]
			FROM posts
			WHERE thread = $1
			GROUP BY path[1]
			ORDER BY path[1] DESC
			LIMIT $2::TEXT::INTEGER
		)
		ORDER BY path[1] DESC, path
	`
	getPostsDescLimitFlatSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1
		ORDER BY id DESC
		LIMIT $2::TEXT::INTEGER
	`
	getPostsLimitTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 
		ORDER BY path
		LIMIT $2::TEXT::INTEGER
	`
	getPostsLimitParentTreeSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 AND path[1] IN (
			SELECT path[1] 
			FROM posts 
			WHERE thread = $1 
			GROUP BY path[1]
			ORDER BY path[1]
			LIMIT $2::TEXT::INTEGER
		)
		ORDER BY path
	`
	getPostsLimitFlatSQL = `
		SELECT id, author, parent, message, forum, thread, created
		FROM posts
		WHERE thread = $1 
		ORDER BY id
		LIMIT $2::TEXT::INTEGER
	`
	// ThreadVote
	getThreadVoteByIDSQL = `
		SELECT votes.voice, threads.id, threads.votes, u.nickname
		FROM (SELECT 1) s
		LEFT JOIN threads ON threads.id = $1
		LEFT JOIN "users" u ON u.nickname = $2
		LEFT JOIN votes ON threads.id = votes.thread AND u.nickname = votes.nickname
	`
	getThreadVoteBySlugSQL = `
		SELECT votes.voice, threads.id, threads.votes, u.nickname
		FROM (SELECT 1) s
		LEFT JOIN threads ON threads.slug = $1
		LEFT JOIN users as u ON u.nickname = $2
		LEFT JOIN votes ON threads.id = votes.thread AND u.nickname = votes.nickname
	`
	insertVoteSQL = `
		INSERT INTO votes (thread, nickname, voice) 
		VALUES ($1, $2, $3)
	`
	updateVoteSQL = `
		UPDATE votes 
		SET voice = $3
		WHERE thread = $1 AND nickname = $2
	`
	updateThreadVotesSQL = `
		UPDATE threads 
		SET	votes = $1
		WHERE id = $2
		RETURNING author, created, forum, "message", slug, title, id, votes
	`
)

func isNumber(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	return false
}

type ThreadDBRepositoryImpl struct{
	db *pgx.ConnPool
	forums ForumRepository
}

func (t *ThreadDBRepositoryImpl) MakeThreadVoteDB(vote *models.Vote, param string) (*models.Thread, error) {
	var err error

	tx, txErr := t.db.Begin()
	if txErr != nil {
		return nil, txErr
	}
	defer tx.Rollback()

	var thread models.Thread
	if isNumber(param) {
		id, _ := strconv.Atoi(param)
		err = tx.QueryRow(`SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id = $1`, id).Scan(
			&thread.ID,
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		)
	} else {
		err = tx.QueryRow(`SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug = $1`, param).Scan(
			&thread.ID,
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		)
	}
	if err != nil {
		return nil, models.ForumNotFound
	}

	var nick string
	err = tx.QueryRow(`SELECT nickname FROM users WHERE nickname = $1`, vote.Nickname).Scan(&nick)
	if err != nil {
		return nil, models.UserNotFound
	}

	rows, err := tx.Exec(`UPDATE votes SET voice = $1 WHERE thread = $2 AND nickname = $3;`, vote.Voice, thread.ID, vote.Nickname)
	if rows.RowsAffected() == 0 {
		_, err := tx.Exec(`INSERT INTO votes (nickname, thread, voice) VALUES ($1, $2, $3);`, vote.Nickname, thread.ID, vote.Voice)
		if err != nil {
			return nil, models.UserNotFound
		}
	}
	// если возник вопрос - в какой мемент делаем +1 к voice -> смотри init.sql

	err = tx.QueryRow(`SELECT votes FROM threads WHERE id = $1`, thread.ID).Scan(&thread.Votes)
	if err != nil {
		return nil, err
	}

	tx.Commit()

	return &thread, nil
}

func (t *ThreadDBRepositoryImpl) GetThread(param string) (*models.Thread, error) {
	var err error
	var thread models.Thread

	if isNumber(param) {
		id, _ := strconv.Atoi(param)
		err = t.db.QueryRow(
			getThreadIdSQL,
			id,
		).Scan(
			&thread.ID,
			&thread.Title,
			&thread.Author,
			&thread.Forum,
			&thread.Message,
			&thread.Votes,
			&thread.Slug,
			&thread.Created,
		)
	} else {
		err = t.db.QueryRow(
			getThreadSlugSQL,
			param,
		).Scan(
			&thread.ID,
			&thread.Title,
			&thread.Author,
			&thread.Forum,
			&thread.Message,
			&thread.Votes,
			&thread.Slug,
			&thread.Created,
		)
	}

	if err != nil {
		return nil, models.ThreadNotFound
	}

	return &thread, nil
}

func (t *ThreadDBRepositoryImpl) Create(thread *models.Thread) (*models.Thread, error) {
	if thread.Slug != "" {
		thread, err := t.GetThread(thread.Slug)
		if err == nil {
			return thread, models.ThreadIsExist
		}
	}


	err := t.db.QueryRow(
		createForumThreadSQL,
		&thread.Author,
		&thread.Created,
		&thread.Message,
		&thread.Title,
		&thread.Slug,
		&thread.Forum,
	).Scan(
		&thread.Author,
		&thread.Created,
		&thread.Forum,
		&thread.ID,
		&thread.Message,
		&thread.Title,
	)

	fmt.Println(thread.ID, err)

	switch ErrorCode(err) {
	case models.PgxOK:
		return thread, nil
	case models.PgxErrNotNull:
		return nil, models.ForumOrAuthorNotFound //UserNotFound
	case models.PgxErrForeignKey:
		return nil, models.ForumOrAuthorNotFound //ForumIsExist
	default:
		return nil, err
	}
}

func (t *ThreadDBRepositoryImpl) UpdateThreadDB(thread *models.ThreadUpdate, param string) (*models.Thread, error) {
	threadFound, err := t.GetThread(param)
	if err != nil {
		return nil, models.PostNotFound
	}

	updatedThread := models.Thread{}

	err = t.db.QueryRow(updateThreadSQL,
		&threadFound.Slug,
		&thread.Title,
		&thread.Message,
	).Scan(
		&updatedThread.ID,
		&updatedThread.Title,
		&updatedThread.Author,
		&updatedThread.Forum,
		&updatedThread.Message,
		&updatedThread.Votes,
		&updatedThread.Slug,
		&updatedThread.Created,
	)

	if err != nil {
		return nil, err
	}

	return &updatedThread, nil
}


func (t *ThreadDBRepositoryImpl) GetThreadsByForum(slug, limit, since, desc string) (*models.Threads, error) {
	var rows *pgx.Rows
	var err error

	if since != "" {
		query := QueryForumWithSince[desc]
		rows, err = t.db.Query(query, slug, since, limit)
	} else {
		query := QueryForumNoSince[desc]
		rows, err = t.db.Query(query, slug, limit)
	}
	defer rows.Close()

	if err != nil {
		return nil, models.ForumNotFound
	}

	threads := models.Threads{}
	for rows.Next() {
		t := models.Thread{}
		err = rows.Scan(
			&t.Author,
			&t.Created,
			&t.Forum,
			&t.ID,
			&t.Message,
			&t.Slug,
			&t.Title,
			&t.Votes,
		)
		threads = append(threads, &t)
	}

	if len(threads) == 0 {
		_, err := t.forums.GetForumBySlug(slug)
		if err != nil {
			return nil, models.ForumNotFound
		}
	}
	return &threads, nil
}

func (t *ThreadDBRepositoryImpl) GetThreadByID(id int64) (*models.Thread, *models.Error) {
	thread := &models.Thread{}
	// спринтф затратно, потом надо это ускорить
	row := t.db.QueryRow(`SELECT thread.id, thread.slug, thread.title, thread.message, thread.votes, thread.created,
									thread.author, thread.forum FROM threads thread WHERE id = $1`, id)
	if err := row.Scan(&thread.ID, &thread.Slug,
		&thread.Title, &thread.Message, &thread.Votes,
		&thread.Created, &thread.Author, &thread.Forum); err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.NewError(models.RowNotFound, "row does not found")
		}

		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	return thread, nil
}

const postID = `
	SELECT id
	FROM posts
	WHERE id = $1 AND thread IN (SELECT id FROM threads WHERE thread <> $2)
`



func NewThreadDBRepositoryImpl(db *pgx.ConnPool, forums ForumRepository) ThreadDBRepository {
	return &ThreadDBRepositoryImpl{db: db,forums:forums}
}

