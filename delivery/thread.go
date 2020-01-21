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

type ThreadHandlers struct {
	threads repository.ThreadDBRepository
}

func NewThreadHandlers(threads repository.ThreadDBRepository) *ThreadHandlers {
	return &ThreadHandlers{threads: threads}
}

func(h *ThreadHandlers) CreateThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	thread := &models.Thread{}
	err = thread.UnmarshalJSON(body)
	thread.Forum = slug

	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}

	result, err := h.threads.Create(thread)

	switch err {
	case nil:
		resp, _ := swag.WriteJSON(result)
		utils.MakeResponse(w, 201, resp)
	case models.ForumOrAuthorNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorUser(slug)))
	case models.ThreadIsExist:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 409, resp)
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

func(h *ThreadHandlers) UpdateThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	param := params["slug_or_id"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	threadUpdate := &models.ThreadUpdate{}
	err = threadUpdate.UnmarshalJSON(body)

	//err = forum.Validate()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}

	result, err := h.threads.UpdateThreadDB(threadUpdate, param)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.PostNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorThread(param)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

func(h *ThreadHandlers) GetThreadsByForum(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug"]
	queryParams := r.URL.Query()
	var limit, since, desc string
	if limit = queryParams.Get("limit"); limit == "" {
		limit = "1"
	}
	since = queryParams.Get("since");
	if since = queryParams.Get("since"); since == "" {
		since = ""
	}
	if desc = queryParams.Get("desc"); desc == ""{
		desc = "false"
	}

	result, err := h.threads.GetThreadsByForum(slug, limit, since, desc)
	switch err {
	case nil:
		resp, _ := swag.WriteJSON(result)
		utils.MakeResponse(w, 200, resp)
	case models.ForumNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorForum(slug)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

func(h *ThreadHandlers) Vote(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	param := params["slug_or_id"]
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	vote := &models.Vote{}
	err = vote.UnmarshalJSON(body)

	result, err := h.threads.MakeThreadVoteDB(vote, param)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.ForumNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorThread(param)))
	case models.UserNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorUser(param)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

func(h *ThreadHandlers) GetThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	param := params["slug_or_id"]

	result, err := h.threads.GetThread(param)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.ThreadNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorThread(param)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}
