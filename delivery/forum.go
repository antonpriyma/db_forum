package delivery

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/AntonPriyma/db_forum/repository"
	"github.com/AntonPriyma/db_forum/utils"
	"github.com/go-openapi/swag"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

type ForumHandlers struct {
	forums repository.ForumRepository
	users repository.UsersRepository
}

func NewForumHandlers(forums repository.ForumRepository, users repository.UsersRepository) *ForumHandlers {
	return &ForumHandlers{forums: forums, users: users}
}

// GetForum получение информации о форуме
func(h *ForumHandlers) GetForum(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug"]

	result, err := h.forums.GetForumBySlug(slug)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.ForumNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorForum(slug)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

// CreateForum создание нового пользователя в базе данных.
func(h *ForumHandlers) CreateForum(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	forum := &models.Forum{}
	err = forum.UnmarshalJSON(body)

	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}

	result, err := h.forums.Create(forum)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 201, resp)
	case models.UserNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorUser(forum.Owner)))
	case models.ForumIsExist:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 409, resp)
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

func(h *ForumHandlers) GetForumUsers(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug"]
	queryParams := r.URL.Query()
	var limit, since, desc string
	if limit = queryParams.Get("limit"); limit == "" {
		limit = "1"
	}
	since = queryParams.Get("since");
	// if since = queryParams.Get("since"); since == "" {
	// 	since = "";
	// }
	if desc = queryParams.Get("desc"); desc == ""{
		desc = "false"
	}

	result, err := h.forums.GetForumUsersDB(slug, limit, since, desc)

	switch err {
	case nil:
		resp, _ := swag.WriteJSON(result) // можно через easyjson, но мне лень было
		utils.MakeResponse(w, 200, resp)
	case models.ForumNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorUser(slug)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}
