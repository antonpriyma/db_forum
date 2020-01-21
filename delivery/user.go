package delivery

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/AntonPriyma/db_forum/repository"
	"github.com/AntonPriyma/db_forum/utils"
	"github.com/go-openapi/swag"
	"io/ioutil"
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
	params := mux.Vars(r)
	nickname := params["nickname"]

	result, err := h.users.GetUserByNickname(nickname)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.UserNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorUser(nickname)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

// CreateUser создание нового пользователя в базе данных.
func(h *UsersHandlers)CreateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	user := &models.User{}
	err = user.UnmarshalJSON(body)
	user.Nickname = nickname

	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	result, err := h.users.Create(user)

	switch err {
	case nil:
		resp, _ := swag.WriteJSON(user)
		utils.MakeResponse(w, 201, resp)
	case models.UserIsExist:
		resp, _ := swag.WriteJSON(result)
		utils.MakeResponse(w, 409, resp)
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

// UpdateUser изменение информации в профиле пользователя.
func(h *UsersHandlers)UpdateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	user := &models.User{}
	err = user.UnmarshalJSON(body)
	user.Nickname = nickname

	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	err = h.users.Save(user)

	switch err {
	case nil:
		resp, _ := user.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.UserNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorUser(nickname)))
	case models.UserUpdateConflict:
		utils.MakeResponse(w, 409, []byte(utils.MakeErrorEmail(nickname)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}
