package repository

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/jackc/pgx"
)

type UsersRepository interface {
	Create(user *models.User) (models.Users, error)
	Save(user *models.User) error
	GetUserByNickname(nickname string) (*models.User, error)
	authorExists(nickname string) bool
}

const (
	createUserSQL = `
		INSERT
		INTO users ("nickname", "fullname", "email", "about")
		VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING
	`
	getUserByNicknameOrEmailSQL = `
		SELECT "nickname", "fullname", "email", "about"
		FROM users
		WHERE "nickname" = $1 OR "email" = $2
	`
	getUserByNickname = `
		SELECT "nickname", "fullname", "email", "about"
		FROM users
		WHERE "nickname" = $1
	`
	getUserSQL = `
		SELECT "nickname", "fullname", "email", "about"
		FROM users
		WHERE "nickname" = $1
	`
	updateUserSQL = `
		UPDATE users
		SET fullname = coalesce(nullif($2, ''), fullname),
			email = coalesce(nullif($3, ''), email),
			about = coalesce(nullif($4, ''), about)
		WHERE "nickname" = $1
		RETURNING nickname, fullname, email, about
	`
)

type UsersRepositoryImpl struct {
	db     *pgx.ConnPool
	forums ForumRepository
}

func NewUsersRepositoryImpl(db *pgx.ConnPool, forums ForumRepository) UsersRepository {
	return &UsersRepositoryImpl{db: db, forums: forums}
}

func (u *UsersRepositoryImpl) Create(user *models.User) (models.Users, error) {
	rows, err := u.db.Exec(
		createUserSQL,
		&user.Nickname,
		&user.Fullname,
		&user.Email,
		&user.About,
	)
	if err != nil {
		return nil, err
	}

	if rows.RowsAffected() == 0 { // пользователь уже есть
		users := models.Users{}
		queryRows, err := u.db.Query(getUserByNicknameOrEmailSQL, user.Nickname, user.Email)
		defer queryRows.Close()

		if err != nil {
			return nil, err
		}

		for queryRows.Next() {
			user := models.User{}
			queryRows.Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About)
			users = append(users, &user)
		}
		return users, models.UserIsExist
	}

	return nil, nil

}

func (u *UsersRepositoryImpl) Save(user *models.User) error {
	err := u.db.QueryRow(
		updateUserSQL,
		&user.Nickname,
		&user.Fullname,
		&user.Email,
		&user.About,
	).Scan(
		&user.Nickname,
		&user.Fullname,
		&user.Email,
		&user.About,
	)

	if err != nil {
		if ErrorCode(err) != models.PgxOK {
			return models.UserUpdateConflict
		}
		return models.UserNotFound
	}

	return nil
}

func (u *UsersRepositoryImpl) GetUserByNickname(nickname string) (*models.User, error) {
	user := models.User{}

	err := GetDB().QueryRow(getUserSQL, nickname).Scan(
		&user.Nickname,
		&user.Fullname,
		&user.Email,
		&user.About,
	)

	if err != nil {
		return nil, models.UserNotFound
	}

	return &user, nil
}

func (u *UsersRepositoryImpl) authorExists(nickname string) bool {
	var user models.User
	err := u.db.QueryRow(
		getUserByNickname,
		nickname,
	).Scan(
		&user.Nickname,
		&user.Fullname,
		&user.About,
		&user.Email,
	)

	if err != nil && err.Error() == models.NoRowsInResult {
		return true
	}
	return false
}
