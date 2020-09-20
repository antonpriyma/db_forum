package repository

import (
	"fmt"
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
	Create(posts *models.Posts, param string) (*models.Posts, error)
	Update(postUpdate *models.PostUpdate, id int) (*models.Post, error)
	GetPostByID(id int, related []string) (*models.PostFull, error)
	GetThreadPostsDB(param, limit, since, sort, desc string) (*models.Posts, error)
}

type PostDBRepositoryImpl struct {
	users  UsersRepository
	thread ThreadDBRepository
	forum ForumRepository
	db     *pgx.ConnPool
}

func(p *PostDBRepositoryImpl) checkPost(post *models.Post, t *models.Thread) error {
	if p.users.authorExists(post.Author) {
		return models.UserNotFound
	}
	if p.thread.parentExitsInOtherThread(post.Parent, t.ID) || p.thread.parentNotExists(post.Parent) {
		return models.PostParentNotFound
	}
	return nil
}

func (p *PostDBRepositoryImpl) Create(posts *models.Posts, param string) (*models.Posts, error) {
	thread, err := p.thread.GetThread(param)
	if err != nil {
		return nil, err
	}

	postsNumber := len(*posts)
	if postsNumber == 0 {
		return posts, nil
	}

	dateTimeTemplate := "2006-01-02 15:04:05"
	created := time.Now().Format(dateTimeTemplate)
	query := strings.Builder{}
	query.WriteString("INSERT INTO posts (author, created, message, thread, parent, forum, path) VALUES ")
	queryBody := "('%s', '%s', '%s', %d, %d, '%s', (SELECT path FROM posts WHERE id = %d) || (SELECT last_value FROM posts_id_seq)),"
	for i, post := range *posts {
		err = p.checkPost(post, thread)
		if err != nil {
			return nil, err
		}

		temp := fmt.Sprintf(queryBody, post.Author, created, post.Message, thread.ID, post.Parent, thread.Forum, post.Parent)
		// удаление запятой в конце queryBody для последнего подзапроса
		if i == postsNumber-1 {
			temp = temp[:len(temp)-1]
		}
		query.WriteString(temp)
	}
	query.WriteString("RETURNING author, created, forum, id, message, parent, thread")

	tx, txErr := p.db.Begin()
	if txErr != nil {
		return nil, txErr
	}
	defer tx.Rollback()

	rows, err := tx.Query(query.String())
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	insertPosts := models.Posts{}
	for rows.Next() {
		post := models.Post{}
		rows.Scan(
			&post.Author,
			&post.Created,
			&post.Forum,
			&post.ID,
			&post.Message,
			&post.Parent,
			&post.Thread,
		)
		insertPosts = append(insertPosts, &post)
	}
	err = rows.Err()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	tx.Commit()
	// по хорошему это впихнуть в хранимые процедуры, но нормальные ребята предпочитают костылить
	p.db.Exec(`UPDATE forums SET posts = posts + $1 WHERE slug = $2`, len(insertPosts), thread.Forum)

	for _, post := range insertPosts {
		p.db.Exec(`  INSERT INTO forum_users ("forum_user", "forum", "email", "fullname", "about")
		SELECT nickname, $2, email, fullname, about
		FROM users
		WHERE nickname = $1
		ON CONFLICT DO NOTHING;`, post.Author, post.Forum)
	}

	return &insertPosts, nil
}

var queryPostsWithSience = map[string]map[string]string{
	"true": map[string]string{
		"tree":        getPostsSienceDescLimitTreeSQL,
		"parent_tree": getPostsSienceDescLimitParentTreeSQL,
		"flat":        getPostsSienceDescLimitFlatSQL,
	},
	"false": map[string]string{
		"tree":        getPostsSienceLimitTreeSQL,
		"parent_tree": getPostsSienceLimitParentTreeSQL,
		"flat":        getPostsSienceLimitFlatSQL,
	},
}

var queryPostsNoSience = map[string]map[string]string{
	"true": map[string]string{
		"tree":        getPostsDescLimitTreeSQL,
		"parent_tree": getPostsDescLimitParentTreeSQL,
		"flat":        getPostsDescLimitFlatSQL,
	},
	"false": map[string]string{
		"tree":        getPostsLimitTreeSQL,
		"parent_tree": getPostsLimitParentTreeSQL,
		"flat":        getPostsLimitFlatSQL,
	},
}

func (p *PostDBRepositoryImpl) Update(postUpdate *models.PostUpdate, id int) (*models.Post, error) {
	post, err := p.GetPostDB(id)
	if err != nil {
		return nil, models.PostNotFound
	}

	if len(postUpdate.Message) == 0 {
		return post, nil
	}

	rows := p.db.QueryRow(updatePostSQL, strconv.Itoa(id), &postUpdate.Message)

	err = rows.Scan(
		&post.Author,
		&post.Created,
		&post.Forum,
		&post.IsEdited,
		&post.Thread,
		&post.Message,
		&post.Parent,
	)

	if err == nil {
		return post, nil
	} else if (err.Error() == noRowsInResult) {
		return nil, models.PostNotFound
	} else {
		return nil, err
	}
}

const noRowsInResult 		= "no rows in result set"


func(p *PostDBRepositoryImpl) GetPostDB(id int) (*models.Post, error) {
	post := models.Post{}

	err := p.db.QueryRow(
		getPostSQL,
		id,
	).Scan(
		&post.ID,
		&post.Author,
		&post.Message,
		&post.Forum,
		&post.Thread,
		&post.Created,
		&post.IsEdited,
		&post.Parent,
	)

	if err == nil {
		return &post, nil
	} else if err.Error() == noRowsInResult {
		return nil, models.PostNotFound
	} else {
		return nil, err
	}
}

func (p *PostDBRepositoryImpl) GetPostByID(id int, related []string) (*models.PostFull, error) {
	postFull := models.PostFull{}
	var err error
	postFull.Post, err = p.GetPostDB(id)
	if err != nil {
		return nil, err
	}

	for _, model := range related {
		switch model {
		case "thread":
			postFull.Thread, err = p.thread.GetThread(strconv.Itoa(int(postFull.Post.Thread)))
		case "forum":
			postFull.Forum, err = p.forum.GetForumBySlug(postFull.Post.Forum)
		case "user":
			postFull.Author, err = p.users.GetUserByNickname(postFull.Post.Author)
		}

		if err != nil {
			return nil, err
		}
	}

	return &postFull, nil
}

func (p *PostDBRepositoryImpl) GetThreadPostsDB(param, limit, since, sort, desc string) (*models.Posts, error) {
	thread, err := p.thread.GetThread(param)
	if err != nil {
		return nil, models.ForumNotFound
	}

	var rows *pgx.Rows

	if since != "" {
		query := queryPostsWithSience[desc][sort]
		rows, err = p.db.Query(query, thread.ID, since, limit)
	} else {
		query := queryPostsNoSience[desc][sort]
		rows, err = p.db.Query(query, thread.ID, limit)
	}
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	posts := models.Posts{}
	for rows.Next() {
		post := models.Post{}

		err = rows.Scan(
			&post.ID,
			&post.Author,
			&post.Parent,
			&post.Message,
			&post.Forum,
			&post.Thread,
			&post.Created,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return &posts, nil
}

func NewPostDBRepositoryImpl(users UsersRepository, thread ThreadDBRepository,forum ForumRepository, db *pgx.ConnPool) PostRepository {
	return &PostDBRepositoryImpl{users: users, thread: thread,forum:forum, db: db}
}
