package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
	"strconv"
	"strings"
	"time"
)

type ThreadDBRepository interface {
	Create(thread *models.Thread) (*models.Thread,*models.Error)
	Update(thread *models.Thread) *models.Error
	VoteBySlug(slug string, voice *models.Vote) (*models.Thread, *models.Error)
	VoteByID(id int64, voice *models.Vote) (*models.Thread,  *models.Error)
	vote(thread *models.Thread,voice *models.Vote)  *models.Error
	GetThreadsByForum(forumSlug string, limit int, since time.Time, desc bool) (models.Threads,  *models.Error)
	GetThreadByID(id int64) (*models.Thread, *models.Error)
	GetThreadBySlug(slug *string) (*models.Thread,  *models.Error)
}

type ThreadDBRepositoryImpl struct{
	db *pgx.ConnPool
}

func (t *ThreadDBRepositoryImpl) Create(thread *models.Thread) (*models.Thread, *models.Error) {
	if validateError := thread.Validate(); validateError != nil {
		return nil, validateError
	}

	tx, err := t.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "can not open 'thread create' transaction")
	}
	defer tx.Rollback()

	duplicate, _ := t.GetThreadBySlug(thread.Slug)
	if duplicate != nil {
		return duplicate, nil
	}

	newRow, err := tx.Query(`INSERT INTO threads (slug, title, message, created, author, forum)  VALUES ($1, $2, $3, $4, $5, (SELECT slug FROM forums WHERE slug = $6)) RETURNING id, forum`,
		thread.Slug, thread.Title, thread.Message, thread.Created, thread.Author, thread.Forum)
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	if !newRow.Next() {
		if pgerr, ok := newRow.Err().(pgx.PgError); ok && (pgerr.Code == "23502" || pgerr.Code == "23503") {
			return nil, models.NewError(models.ForeignKeyNotFound, pgerr.Error())
		}

		return nil, models.NewError(models.InternalDatabase, newRow.Err().Error())
	}
	if err = newRow.Scan(&thread.ID, &thread.Forum); err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	newRow.Close()

	if err = tx.Commit(); err != nil {
		return nil, models.NewError(models.InternalDatabase, "thread create transaction commit error")
	}

	return nil, nil
}

func (t *ThreadDBRepositoryImpl) Update(threadInput *models.Thread) *models.Error {
	tx, err := t.db.Begin()
	if err != nil {
		return models.NewError(models.InternalDatabase, "can not open 'thread update' transaction")
	}
	defer tx.Rollback()

	var thread *models.Thread
	if threadInput.Slug == nil {
		thread, _ = t.GetThreadByID(threadInput.ID)
	} else {
		thread, _ = t.GetThreadBySlug(threadInput.Slug)
	}
	if thread == nil {
		return models.NewError(models.RowNotFound, "can not find thread with this fields")
	}

	needsUpdate := false
	if threadInput.Message != "" && threadInput.Message != thread.Message {
		needsUpdate = true
		thread.Message = threadInput.Message
	}
	if threadInput.Title != "" && threadInput.Title != thread.Title {
		needsUpdate = true
		thread.Title = threadInput.Title
	}
	*threadInput = *thread

	// нечего обновлять
	if !needsUpdate {
		return nil
	}

	_, err = tx.Exec(`UPDATE threads SET (message, title) = ($1, $2) WHERE id = $3`, threadInput.Message, threadInput.Title, threadInput.ID)
	if err != nil {
		return models.NewError(models.InternalDatabase, err.Error())
	}

	if err = tx.Commit(); err != nil {
		return models.NewError(models.InternalDatabase, "thread update transaction commit error")
	}

	return nil
}

func (t *ThreadDBRepositoryImpl) VoteBySlug(slug string, voice *models.Vote) (*models.Thread, *models.Error) {
	voice.VoiceImpl = (voice.Voice == 1)

	tx, err := t.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "can not open 'thread vode' transaction")
	}
	defer tx.Rollback()

	thread, _ := t.GetThreadBySlug(&slug)
	if thread == nil {
		return nil, models.NewError(models.RowNotFound, "no thread with this slug")
	}

	voteError := t.vote(thread,voice)
	if voteError != nil {
		return nil, voteError
	}

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "vote transaction commit error")
	}

	return thread, nil
}

func (t *ThreadDBRepositoryImpl) VoteByID(id int64, voice *models.Vote) (*models.Thread, *models.Error) {
	voice.VoiceImpl = (voice.Voice == 1)

	tx, err := t.db.Begin()
	if err != nil {
		return nil,models.NewError(models.InternalDatabase, "can not open 'thread vode' transaction")
	}
	defer tx.Rollback()

	//formatedID := strconv.FormatInt(id, 10)
	thread, _ := t.GetThreadByID(id)
	if thread == nil {
		return nil, models.NewError(models.RowNotFound, "no thread with this slug")
	}

	voteError := t.vote(thread, voice)
	if voteError != nil {
		return nil, voteError
	}

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "vote transaction commit error")
	}

	return thread, nil
}

func (t *ThreadDBRepositoryImpl) vote(thread *models.Thread, voice *models.Vote) *models.Error {
	// чтобы далее понять, обновилось ли что-то
	oldVotes := thread.Votes

	var isUp bool
	row := t.db.QueryRow(`SELECT v.is_up FROM votes v WHERE author = $1 AND thread = $2`, voice.Nickname, thread.ID)
	if err := row.Scan(&isUp); err == nil {
		// нужно обновить
		if isUp != voice.VoiceImpl {
			_, err = t.db.Exec(`UPDATE votes SET is_up = $1 WHERE author = $2 AND thread = $3`, voice.VoiceImpl, voice.Nickname, thread.ID)
			if err != nil {
				return models.NewError(models.InternalDatabase, err.Error())
			}

			// затираем старый + добавили новый
			thread.Votes += 2 * int32(voice.Voice)
		}
	} else if err == pgx.ErrNoRows {
		_, err = t.db.Exec(`INSERT INTO votes (author, thread, is_up) VALUES ($1, $2, $3)`, voice.Nickname, thread.ID, voice.VoiceImpl)
		if err != nil {
			if pgerr, ok := err.(pgx.PgError); ok && pgerr.Code == "23503" {
				return models.NewError(models.ForeignKeyNotFound, pgerr.Error())
			}

			return models.NewError(models.InternalDatabase, err.Error())
		}

		thread.Votes += int32(voice.Voice)
	} else {
		return models.NewError(models.InternalDatabase, err.Error())
	}

	if oldVotes != thread.Votes {
		_, err := t.db.Exec(`UPDATE threads SET votes = $1 WHERE id = $2`, thread.Votes, thread.ID)
		if err != nil {
			return models.NewError(models.InternalDatabase, err.Error())
		}
	}

	return nil
}

func (t *ThreadDBRepositoryImpl) GetThreadsByForum(forumSlug string, limit int, since time.Time, desc bool) (models.Threads, *models.Error) {
	query := strings.Builder{}
	args := []interface{}{forumSlug}
	query.WriteString(`SELECT t.id, t.slug, t.title, t.message, t.votes, t.created,
						t.author, t.forum FROM threads t WHERE forum = $1`)
	if !since.IsZero() {
		query.WriteString(" AND created")
		if desc {
			query.WriteString(" <=")
		} else {
			query.WriteString(" >=")
		}
		query.WriteString(" $2")
		args = append(args, since)
	}
	query.WriteString(" ORDER BY t.created")
	if desc {
		query.WriteString(" DESC")
	}
	if limit != -1 {
		query.WriteString(" LIMIT $")
		query.WriteString(strconv.Itoa(len(args) + 1))
		args = append(args, limit)
	}
	query.WriteByte(';')

	tx, err := t.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "can not open 'thread get' transaction")
	}
	defer tx.Rollback()

	forum, _ := getForumBySlugImpl(tx, forumSlug)
	if forum == nil {
		return nil, models.NewError(models.RowNotFound, "no threads for this forum")
	}
	rows, err := tx.Query(query.String(), args...)
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	threads := make([]*models.Thread, 0)
	for rows.Next() {
		t := &models.Thread{}
		err = rows.Scan(&t.ID, &t.Slug,
			&t.Title, &t.Message, &t.Votes,
			&t.Created, &t.Author, &t.Forum)
		if err != nil {
			return nil, models.NewError(models.InternalDatabase, err.Error())
		}
		threads = append(threads, t)
	}
	rows.Close()

	if err = tx.Commit(); err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	return threads, nil
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

func (t *ThreadDBRepositoryImpl) GetThreadBySlug(slug *string) (*models.Thread, *models.Error) {
	thread := &models.Thread{}
	// спринтф затратно, потом надо это ускорить
	row := t.db.QueryRow(`SELECT thread.id, thread.slug, thread.title, thread.message, thread.votes, thread.created,
									thread.author, thread.forum FROM threads thread WHERE slug = $1`, slug)
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

func NewThreadDBRepositoryImpl(db *pgx.ConnPool) ThreadDBRepository {
	return &ThreadDBRepositoryImpl{db: db}
}

