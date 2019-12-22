package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
	"strconv"
	"strings"
	"time"
)


type SortMode int

const (
	Flat SortMode = iota
	Tree
	ParentTree
)

type PostRepository interface {
	Create(ps models.Posts,slugOrID interface{}) *models.Error
	Update(post *models.Post) *models.Error
	GetPostByID(id int64, scope []string) (*models.PostFull, *models.Error)
	GetPostsByThreadID(threadID int64, limit int, since int64, mode SortMode, desc bool) (models.Posts, *models.Error)
	GetPostsByThreadSlug(slug string, limit int, since int64, mode SortMode, desc bool) (models.Posts, *models.Error)
}

type PostDBRepositoryImpl struct {
	users UsersRepository
	thread ThreadDBRepository
	db *pgx.ConnPool
}

func NewPostDBRepositoryImpl(users UsersRepository, thread ThreadDBRepository, db *pgx.ConnPool) PostRepository {
	return &PostDBRepositoryImpl{users: users, thread: thread, db: db}
}

func (p *PostDBRepositoryImpl) Create(ps models.Posts,slugOrID interface{}) *models.Error {
	tx, err := p.db.Begin()
	if err != nil {
		return models.NewError(models.InternalDatabase, "can not open posts create tx")
	}
	defer tx.Rollback()

	var thread *models.Thread
	switch v := slugOrID.(type) {
	case int64:
		thread, _ = p.thread.GetThreadByID(v)
	case string:
		thread, _ = p.thread.GetThreadBySlug(&v)
	}
	if thread == nil {
		return models.NewError(models.ForeignKeyNotFound, "thread not found")
	}

	var createError *models.Error
	for _, post := range ps {
		createError = p.createImpl(post, thread)
		if createError != nil {
			return createError
		}
	}

	err = tx.Commit()
	if err != nil {
		return models.NewError(models.InternalDatabase, "con not commit posts create tx")
	}

	return nil
}

func (r *PostDBRepositoryImpl) createImpl(p *models.Post,thread *models.Thread) *models.Error {
	if validateError := p.Validate(); validateError != nil {
		return validateError
	}

	p.Forum = thread.Forum // можно убрать поле форум в табоице posts
	p.Thread = thread.ID

	if p.Parent != 0 {
		p.ParentImpl = &p.Parent
		row := r.db.QueryRow(`SELECT p.thread FROM posts p WHERE p.id = $1;`, p.ParentImpl)

		var parentThread int64
		if err := row.Scan(&parentThread); err != nil || parentThread != p.Thread {
			return models.NewError(models.ForeignKeyConflict, "Parent post was created in another thread")
		}
	}

	if p.Created.IsZero() {
		p.Created = time.Now()
	}

	// PostgreSQL считает с точностью до MS
	p.Created = p.Created.Round(time.Millisecond)
	newRow, err := r.db.Query(`INSERT INTO posts (message, created, author, forum, thread, parent, parents)
	VALUES ($1, $2, $3, $4, $5, $6, (SELECT parents FROM posts WHERE posts.id = $6) || (SELECT currval('posts_id_seq'))) RETURNING id`,
		p.Message, p.Created, p.Author, p.Forum, p.Thread, p.ParentImpl)
	if err != nil {
		return models.NewError(models.InternalDatabase, err.Error())
	}
	if !newRow.Next() {
		if pgerr, ok := newRow.Err().(pgx.PgError); ok && pgerr.Code == "23503" {
			return models.NewError(models.ForeignKeyNotFound, pgerr.Error())
		}

		return models.NewError(models.InternalDatabase, newRow.Err().Error())
	}
	// обновляем структуру, чтобы она содержала валидное имя создателя(учитывая регистр)
	// и валидный ID
	if err = newRow.Scan(&p.ID); err != nil {
		return models.NewError(models.InternalDatabase, err.Error())
	}
	newRow.Close()

	return nil
}

func (p *PostDBRepositoryImpl) Update(post *models.Post) *models.Error {
	tx, err := p.db.Begin()
	if err != nil {
		return models.NewError(models.InternalDatabase, "can not open 'thread update' transaction")
	}
	defer tx.Rollback()

	storedPost := &models.Post{}
	row := tx.QueryRow(`SELECT p.id, p.message, p.is_edited, p.created, p.author, p.forum, p.thread, p.parent FROM posts p WHERE p.id = $1;`,
		post.ID)
	if err := row.Scan(&storedPost.ID, &storedPost.Message,
		&storedPost.IsEdited, &storedPost.Created, &storedPost.Author,
		&storedPost.Forum, &storedPost.Thread, &storedPost.ParentImpl); err != nil {
		if err == pgx.ErrNoRows {
			return models.NewError(models.RowNotFound, err.Error())
		}

		return models.NewError(models.InternalDatabase, err.Error())
	}
	if storedPost.ParentImpl == nil {
		storedPost.Parent = 0
	} else {
		storedPost.Parent = *storedPost.ParentImpl
	}

	needsUpdate := false
	if post.Message != "" && storedPost.Message != post.Message {
		needsUpdate = true
		storedPost.Message = post.Message
		storedPost.IsEdited = true
	}
	*post = *storedPost

	if !needsUpdate {
		return nil
	}

	_, err = tx.Exec(`UPDATE posts SET (message, is_edited) = ($1, $2) WHERE id = $3`, post.Message, post.IsEdited, post.ID)
	if err != nil {
		return models.NewError(models.InternalDatabase, err.Error())
	}

	if err = tx.Commit(); err != nil {
		return models.NewError(models.InternalDatabase, "thread update transaction commit error")
	}

	return nil
}

func (p *PostDBRepositoryImpl) GetPostByID(id int64, scope []string) (*models.PostFull, *models.Error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	defer tx.Rollback()

	pf := &models.PostFull{}
	var getErr *models.Error

	pf.Post, getErr = p.getPostByIDImpl(id)
	if getErr != nil {
		return nil, getErr
	}

	for _, sc := range scope {
		switch sc {
		case "user":
			pf.Author, getErr = p.users.GetUserByNickname(pf.Post.Author)
		case "forum":
			pf.Forum, getErr = getForumBySlugImpl(tx, pf.Post.Forum)
		case "thread":
			pf.Thread, getErr = p.thread.GetThreadByID(pf.Post.Thread)
		}
		if getErr != nil {
			return nil, getErr
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	return pf, nil
}

func (p *PostDBRepositoryImpl) GetPostsByThreadID(threadID int64, limit int, since int64, mode SortMode, desc bool) (models.Posts, *models.Error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "get posts trans open error")
	}
	defer tx.Rollback()

	res, getError := p.getPostsByThreadIDImpl(threadID, limit, since, mode, desc)
	if getError != nil {
		return nil, getError
	}

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "get posts trans commit error")
	}

	return res, nil
}

func (p *PostDBRepositoryImpl) GetPostsByThreadSlug(slug string, limit int, since int64, mode SortMode, desc bool) (models.Posts, *models.Error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "get posts trans open error")
	}
	defer tx.Rollback()

	thread, _ := p.thread.GetThreadBySlug(&slug)
	if thread == nil {
		return nil, models.NewError(models.RowNotFound, "thread not exists")
	}

	res, getError := p.getPostsByThreadIDImpl(thread.ID, limit, since, mode, desc)
	if getError != nil {
		return nil, getError
	}

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "get posts trans commit error")
	}

	return res, nil
}


func (p *PostDBRepositoryImpl) getPostsByThreadIDImpl(threadID int64, limit int, since int64, mode SortMode, desc bool) (models.Posts,*models.Error) {
	query := strings.Builder{}
	args := []interface{}{}
	switch mode {
	case Flat:
		args = append(args, threadID)
		query.WriteString(`SELECT p.id, p.message, p.is_edited, p.created, p.author,
						 p.forum, p.thread, p.parent FROM posts p WHERE p.thread = $1`)
		if since != 0 {
			args = append(args, since)
			query.WriteString(` AND (p.created, p.id) `)
			if desc {
				query.WriteByte('<')
			} else {
				query.WriteByte('>')
			}
			query.WriteString(` (SELECT posts.created, posts.id FROM posts WHERE posts.id=$2)`)
		}
		query.WriteString(` ORDER BY (p.created, p.id)`)
		if desc {
			query.WriteString(" DESC")
		}
		if limit != -1 {
			query.WriteString(" LIMIT $")
			query.WriteString(strconv.Itoa(len(args) + 1))
			args = append(args, limit)
		}
	case Tree:
		args = append(args, threadID)
		query.WriteString(`SELECT p.id, p.message, p.is_edited, p.created, p.author,
			p.forum, p.thread, p.parent FROM posts p WHERE p.thread = $1`)
		if since != 0 {
			args = append(args, since)
			query.WriteString(" AND p.parents ")
			if desc {
				query.WriteByte('<')
			} else {
				query.WriteByte('>')
			}
			query.WriteString(` (SELECT posts.parents FROM posts WHERE posts.id = $2)`)
		}
		query.WriteString(" ORDER BY p.parents")
		if desc {
			query.WriteString(" DESC")
		}
		if limit != -1 {
			query.WriteString(" LIMIT $")
			query.WriteString(strconv.Itoa(len(args) + 1))
			args = append(args, limit)
		}
	case ParentTree:
		args = append(args, threadID)
		query.WriteString(`SELECT p.id, p.message, p.is_edited, p.created, p.author,
			p.forum, p.thread, p.parent FROM posts p WHERE p.parents[1] IN (
				SELECT posts.id FROM posts WHERE posts.thread = $1 AND posts.parent IS NULL`)
		if since != 0 {
			args = append(args, since)
			query.WriteString(` AND posts.id`)
			if desc {
				query.WriteByte('<')
			} else {
				query.WriteByte('>')
			}
			query.WriteString(` (SELECT COALESCE(posts.parents[1], posts.id) FROM posts WHERE posts.id = $2)`)
		}
		query.WriteString(" ORDER BY posts.id")
		if desc {
			query.WriteString(" DESC")
		}
		if limit != -1 {
			query.WriteString(" LIMIT $")
			query.WriteString(strconv.Itoa(len(args) + 1))
			args = append(args, limit)
		}
		query.WriteString(`) ORDER BY`)
		if desc {
			query.WriteString(` p.parents[1] DESC,`)
		}
		query.WriteString(` p.parents`)
	}
	query.WriteByte(';')


	thread, _ := p.thread.GetThreadByID(threadID)
	if thread == nil {
		return nil, models.NewError(models.RowNotFound, "no posts for this thread")
	}

	rows, err := p.db.Query(query.String(), args...)
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	posts := make([]*models.Post, 0)
	for rows.Next() {
		p := &models.Post{}
		err = rows.Scan(&p.ID, &p.Message,
			&p.IsEdited, &p.Created, &p.Author,
			&p.Forum, &p.Thread, &p.ParentImpl)
		if err != nil {
			return nil, models.NewError(models.InternalDatabase, err.Error())
		}
		if p.ParentImpl == nil {
			p.Parent = 0
		} else {
			p.Parent=*p.ParentImpl
		}

		posts = append(posts, p)
	}
	rows.Close()

	return posts, nil
}

func (p *PostDBRepositoryImpl) getPostByIDImpl(id int64) (*models.Post, *models.Error) {
	post := &models.Post{}

	row := p.db.QueryRow(`SELECT p.id, p.message, p.is_edited, p.created, p.author, p.forum, p.thread, p.parent FROM posts p WHERE p.id = $1;`, id)
	if err := row.Scan(&post.ID, &post.Message,
		&post.IsEdited, &post.Created, &post.Author,
		&post.Forum, &post.Thread, &post.ParentImpl); err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.NewError(models.RowNotFound, err.Error())
		}

		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	if post.ParentImpl == nil {
		post.Parent = 0
	} else {
		post.Parent = *post.ParentImpl
	}

	return post, nil
}




