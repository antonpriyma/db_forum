package delivery

import (
	"encoding/json"
	"github.com/AntonPriyma/db_forum/models"
	"github.com/AntonPriyma/db_forum/repository"
	"github.com/AntonPriyma/db_forum/utils"
	"github.com/go-openapi/swag"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type PostHandlers struct {
	posts repository.PostRepository
	users repository.UsersRepository
}

func NewPostHandlers(posts repository.PostRepository, users repository.UsersRepository) *PostHandlers {
	return &PostHandlers{posts: posts, users: users}
}

func(h *PostHandlers) CreatePosts(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	param := params["slug_or_id"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	posts := &models.Posts{}
	err = json.Unmarshal(body, &posts)
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}

	result, err := h.posts.Create(posts, param)



	switch err {
	case nil:
		resp,_:=swag.WriteJSON(result)
		utils.MakeResponse(w, 201, resp)
	case models.ThreadNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorThreadID(param)))
	case models.UserNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorPostAuthor(param)))
	case models.PostParentNotFound:
		utils.MakeResponse(w, 409, []byte(utils.MakeErrorThreadConflict()))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

func(h *PostHandlers) GetPosts(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	param := params["slug_or_id"]
	queryParams := r.URL.Query()
	var limit, since, sort, desc string
	if limit = queryParams.Get("limit"); limit == "" {
		limit = "1"
	}
	since = queryParams.Get("since");
	// if since = queryParams.Get("since"); since == "" {
	// 	since = "";
	// }
	if sort = queryParams.Get("sort"); sort == ""{
		sort = "flat"
	}
	if desc = queryParams.Get("desc"); desc == ""{
		desc = "false"
	}
	// fmt.Println("limit", limit, "since", since, "sort", sort, "desc", desc)
	result, err := h.posts.GetThreadPostsDB(param, limit, since, sort, desc)

	// resp, _ := result.MarshalJSON()

	switch err {
	case nil:
		resp, _ := swag.WriteJSON(result)
		utils.MakeResponse(w, 200, resp)
	case models.ForumNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorThread(param)))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}

func(h *PostHandlers) GetPost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}

	queryParams := r.URL.Query()
	relatedQuery := queryParams.Get("related")
	var related []string
	related = append(related, strings.Split(relatedQuery, ",")...)

	result, err := h.posts.GetPostByID(id, related)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.PostNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorPost(string(id))))
	default:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorPost(string(id))))
	}
}

func(h *PostHandlers) UpdatePost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	postUpdate := &models.PostUpdate{}
	err = postUpdate.UnmarshalJSON(body)

	if err != nil {
		utils.MakeResponse(w, 500, []byte(err.Error()))
		return
	}
	result, err := h.posts.Update(postUpdate, id)
	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		utils.MakeResponse(w, 200, resp)
	case models.PostNotFound:
		utils.MakeResponse(w, 404, []byte(utils.MakeErrorPost(string(id))))
	default:
		utils.MakeResponse(w, 500, []byte(err.Error()))
	}
}
