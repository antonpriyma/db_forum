package delivery

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/AntonPriyma/db_forum/repository"
	"github.com/AntonPriyma/db_forum/utils"
	"net/http"



	"github.com/gorilla/mux"
)

type UsersHandlers struct {
	users repository.UsersRepository
}

func NewUsersHandlers(users repository.UsersRepository) *UsersHandlers {
	return &UsersHandlers{users: users}
}

// GetUser получение информации о пользователе форума по его имени.
func(h *UsersHandlers) GetUser(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
	userInfo, err := h.users.GetUserByNickname(vars["nickname"])
	if err != nil {
		var code int
		if err.Code == models.InternalDatabase {
			code = http.StatusInternalServerError
		} else if err.Code == models.RowNotFound {
			code = http.StatusNotFound
		}

		utils.WriteEasyjson(w, code, err)
		return
	}

	utils.WriteEasyjson(w, http.StatusOK, userInfo)
}

// CreateUser создание нового пользователя в базе данных.
func(h *UsersHandlers)CreateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	newUser := &models.User{}
	err := utils.DecodeEasyjson(r.Body, newUser)
	if err != nil {
		utils.WriteEasyjson(w, http.StatusBadRequest, &models.Error{
			Message: "unable to decode request body;",
		})
		return
	}

	newUser.Nickname = vars["nickname"]
	used, errs := h.users.Create(newUser)
	if used != nil {
		utils.WriteEasyjson(w, http.StatusConflict, used)
		return
	}

	if errs != nil {
		var code int
		if errs.Code == models.ValidationFailed {
			code = http.StatusBadRequest
		} else {
			code = http.StatusInternalServerError
		}

		utils.WriteEasyjson(w, code, errs)
		return
	}

	utils.WriteEasyjson(w, http.StatusCreated, newUser)
}

// UpdateUser изменение информации в профиле пользователя.
func(h *UsersHandlers)UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	updateFields := &models.UpdateUserFields{}
	jsonErr := utils.DecodeEasyjson(r.Body, updateFields)
	if jsonErr != nil {
		utils.WriteEasyjson(w, http.StatusBadRequest, &models.Error{
			Message: "unable to decode request body;",
		})
		return
	}

	user, err := h.users.GetUserByNickname(vars["nickname"])
	if err != nil {
		var code int
		if err.Code == models.InternalDatabase {
			code = http.StatusInternalServerError
		} else if err.Code == models.RowNotFound {
			code = http.StatusNotFound
		}

		utils.WriteEasyjson(w, code, err)
		return
	}

	if updateFields.Fullname != nil {
		user.Fullname = *updateFields.Fullname
	}
	if updateFields.About != nil {
		user.About = *updateFields.About
	}
	if updateFields.Email != nil {
		user.Email = *updateFields.Email
	}

	err = h.users.Save(user)
	if err != nil {
		var code int
		if err.Code == models.InternalDatabase {
			code = http.StatusInternalServerError
		} else if err.Code == models.RowDuplication {
			code = http.StatusConflict
		}

		utils.WriteEasyjson(w, code, err)
		return
	}

	utils.WriteEasyjson(w, http.StatusOK, user)
}
