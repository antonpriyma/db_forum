package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
	"strconv"
	"strings"
)

type UsersRepository interface {
	Create(user *models.User) (models.Users, *models.Error)
	Save(user *models.User) *models.Error
	GetUserByNickname(nickname string) (*models.User, *models.Error)
	GetUserByEmail(nickname string) (*models.User, *models.Error)
	getDuplicates(username string,email string) models.Users
	GetUsersByForumSlug(slug string, limit int, since string, desc bool) (models.Users, *models.Error)

}

type UsersRepositoryImpl struct{
	db *pgx.ConnPool
	forums ForumRepository
}

func NewUsersRepositoryImpl(db *pgx.ConnPool, forums ForumRepository) UsersRepository {
	return &UsersRepositoryImpl{db: db, forums: forums}
}

func (u *UsersRepositoryImpl) getDuplicates(username string, email string) (models.Users) {
	usedUsers := make([]*models.User, 0)

	dupNickname, _ :=u.GetUserByNickname(username)
	if dupNickname != nil {
		usedUsers = append(usedUsers, dupNickname)
	}

	dupEmail, _ := u.GetUserByEmail(email)
	if dupEmail != nil && (dupNickname == nil || dupEmail.ID != dupNickname.ID) {
		usedUsers = append(usedUsers, dupEmail)
	}

	if len(usedUsers) == 0 {
		return nil
	}

	return usedUsers
}

func (u *UsersRepositoryImpl) Create(user *models.User) (models.Users, *models.Error) {
	if validateError := user.Validate(); validateError != nil {
		return nil, validateError
	}

	tx, err := u.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "can not open 'user create' transaction")
	}
	defer tx.Rollback()

	// валдация на повторы
	usedUsers := u.getDuplicates(user.Nickname, user.Email)
	if usedUsers != nil {
		return usedUsers, models.NewError(models.RowDuplication, "email or nickname are already used!")
	}

	newRow, err := tx.Query(`INSERT INTO users (nickname, fullname, about, email) VALUES ($1, $2, $3, $4) RETURNING id`,
		user.Nickname, user.Fullname, user.About, user.Email)
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}
	if !newRow.Next() {
		return nil, models.NewError(models.RowNotFound, "row does not found")
	}

	// обновляем структуру так, чтобы она содержала валидный id
	if err = newRow.Scan(&user.ID); err != nil {
		return nil, models.NewError(models.RowNotFound, "row does not found")
	}
	newRow.Close()

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "user create transaction commit error")
	}

	return nil, nil

}



func (u *UsersRepositoryImpl) Save(user *models.User) *models.Error {
	if user.ID == 0 {
		return models.NewError(models.ValidationFailed, "ID must be setted")
	}

	if err := user.Validate(); err != nil {
		return err
	}

	// возможно далее указывать в запросе не все поля
	_, err :=u.db.Exec(`UPDATE users SET (nickname, fullname, about, email) = ($1, $2, $3, $4) WHERE id = $5`,
		user.Nickname, user.Fullname, user.About, user.Email, user.ID)
	if err != nil {
		if pgerr, ok := err.(pgx.PgError); ok && pgerr.Code == "23505" {
			return models.NewError(models.RowDuplication, pgerr.Error())
		}

		return models.NewError(models.InternalDatabase, err.Error())
	}

	return nil
}

func (u *UsersRepositoryImpl) GetUserByNickname(nickname string) (*models.User, *models.Error) {
	user := &models.User{}
	// спринтф затратно, потом надо это ускорить
	row := *u.db.QueryRow(`SELECT u.id, u.nickname, u.fullname, u.about, u.email FROM users u WHERE nickname = $1`, nickname)
	if err := row.Scan(&user.ID, &user.Nickname,
		&user.Fullname, &user.About, &user.Email); err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.NewError(models.RowNotFound, "row does not found")
		}

		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	return user, nil
}

func (u *UsersRepositoryImpl) GetUserByEmail(nickname string) (*models.User, *models.Error) {
	user := &models.User{}
	// спринтф затратно, потом надо это ускорить
	row := *u.db.QueryRow(`SELECT u.id, u.nickname, u.fullname, u.about, u.email FROM users u WHERE email = $1`, nickname)
	if err := row.Scan(&user.ID, &user.Nickname,
		&user.Fullname, &user.About, &user.Email); err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.NewError(models.RowNotFound, "row does not found "+nickname)
		}

		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	return user, nil
}

func (u *UsersRepositoryImpl) GetUsersByForumSlug(slug string, limit int, since string, desc bool) (models.Users, *models.Error) {
	tx, err := u.db.Begin()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, "can not open 'thread update' transaction")
	}
	defer tx.Rollback()

	forum, _ := u.forums.GetForumBySlug(slug)
	if forum == nil {
		return nil, models.NewError(models.RowNotFound, "forum not found")
	}

	query := strings.Builder{}
	args := []interface{}{slug}
	query.WriteString(`SELECT DISTINCT ON (u.nickname COLLATE "C") u.id, u.nickname, u.fullname, u.about, u.email FROM users u WHERE nickname IN (
		SELECT author FROM threads WHERE forum = $1
	  UNION ALL
	  SELECT author FROM posts WHERE forum = $1
	)`)
	if since != "" {
		args = append(args, since)
		query.WriteString(` AND nickname COLLATE "C" `)
		if desc {
			query.WriteByte('<')
		} else {
			query.WriteByte('>')
		}
		query.WriteString(` $2`)
	}

	query.WriteString(` ORDER BY (u.nickname COLLATE "C")`)
	if desc {
		query.WriteString(" DESC")
	}
	if limit != -1 {
		query.WriteString(" LIMIT $")
		query.WriteString(strconv.Itoa(len(args) + 1))
		args = append(args, limit)
	}
	query.WriteByte(';')

	rows, err := tx.Query(query.String(), args...)
	if err != nil {
		return nil,models.NewError(models.InternalDatabase, err.Error())
	}

	users := make([]*models.User, 0)
	for rows.Next() {
		u := &models.User{}
		err = rows.Scan(&u.ID, &u.Nickname,
			&u.Fullname, &u.About, &u.Email)
		if err != nil {
			return nil, models.NewError(models.InternalDatabase, err.Error())
		}
		users = append(users, u)
	}
	rows.Close()

	err = tx.Commit()
	if err != nil {
		return nil, models.NewError(models.InternalDatabase, err.Error())
	}

	return users, nil
}






