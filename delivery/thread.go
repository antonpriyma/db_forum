package delivery

import (
	"github.com/AntonPriyma/db_forum/models"
	"github.com/AntonPriyma/db_forum/repository"
	"github.com/AntonPriyma/db_forum/utils"
	"net/http"
	"strconv"
	"time"


	"github.com/gorilla/mux"
)

type ThreadHandlers struct {
	threads repository.ThreadDBRepository
}

func NewThreadHandlers(threads repository.ThreadDBRepository) *ThreadHandlers {
	return &ThreadHandlers{threads: threads}
}

// CreateForum создание нового пользователя в базе данных.
func(h *ThreadHandlers) CreateThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	newThread := &models.Thread{}
	err := utils.DecodeEasyjson(r.Body, newThread)
	if err != nil {
		utils.WriteEasyjson(w, http.StatusBadRequest, &models.Error{
			Message: "unable to decode request body;",
		})
		return
	}

	newThread.Forum = vars["slug"]
	used, errs := h.threads.Create(newThread)
	if used != nil {
		utils.WriteEasyjson(w, http.StatusConflict, used)
		return
	}

	if errs != nil {
		var code int
		if errs.Code == models.ValidationFailed {
			code = http.StatusBadRequest
		} else if errs.Code == models.ForeignKeyNotFound {
			code = http.StatusNotFound
		} else {
			code = http.StatusInternalServerError
		}

		utils.WriteEasyjson(w, code, errs)
		return
	}

	utils.WriteEasyjson(w, http.StatusCreated, newThread)
}

func(h *ThreadHandlers) UpdateThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	slug := vars["slug_or_id"]
	threadID, err := strconv.ParseInt(slug, 10, 64)
	isID := (err == nil)

	thread := &models.Thread{}
	err = utils.DecodeEasyjson(r.Body, thread)
	if err != nil {
		utils.WriteEasyjson(w, http.StatusBadRequest, &models.Error{
			Message: "unable to decode request body;",
		})
		return
	}

	if isID {
		thread.ID = threadID
	} else {
		thread.Slug = &slug
	}

	if updErr := h.threads.Update(thread); updErr != nil {
		if updErr.Code == models.RowNotFound {
			utils.WriteEasyjson(w, http.StatusNotFound, updErr)
			return
		}

		utils.WriteEasyjson(w, http.StatusInternalServerError, updErr)
		return
	}

	utils.WriteEasyjson(w, http.StatusOK, thread)
}

func(h *ThreadHandlers) GetThreadsByForum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	query := r.URL.Query()
	limitParam, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		limitParam = -1
	}
	offsetParam, _ := time.Parse(time.RFC3339Nano, query.Get("since"))
	desc := (query.Get("desc") == "true")

	threads, threadsErr := h.threads.GetThreadsByForum(vars["slug"], limitParam, offsetParam, desc)
	if threadsErr != nil {
		if threadsErr.Code == models.RowNotFound {
			utils.WriteEasyjson(w, http.StatusNotFound, threadsErr)
			return
		}

		utils.WriteEasyjson(w, http.StatusInternalServerError, threadsErr)
		return
	}

	utils.WriteEasyjson(w, http.StatusOK, threads)
}

func(h *ThreadHandlers) Vote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	slug := vars["slug_or_id"]
	threadID, err := strconv.ParseInt(slug, 10, 64)
	isID := (err == nil)

	voice := &models.Vote{}
	err = utils.DecodeEasyjson(r.Body, voice)
	if err != nil {
		utils.WriteEasyjson(w, http.StatusBadRequest, &models.Error{
			Message: "unable to decode request body;",
		})
		return
	}

	var thread *models.Thread
	var threadErr *models.Error
	if isID {
		thread, threadErr = h.threads.VoteByID(threadID, voice)
	} else {
		thread, threadErr = h.threads.VoteBySlug(slug, voice)
	}
	if threadErr != nil {
		if threadErr.Code == models.RowNotFound || threadErr.Code == models.ForeignKeyNotFound {
			utils.WriteEasyjson(w, http.StatusNotFound, threadErr)
			return
		}

		utils.WriteEasyjson(w, http.StatusInternalServerError, threadErr)
		return
	}

	utils.WriteEasyjson(w, http.StatusOK, thread)
}

func(h *ThreadHandlers) GetThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	slug := vars["slug_or_id"]
	threadID, err := strconv.ParseInt(slug, 10, 64)
	isID := (err == nil)

	var thread *models.Thread
	var threadErr *models.Error
	if isID {
		thread, threadErr = h.threads.GetThreadByID(threadID)
	} else {
		thread, threadErr = h.threads.GetThreadBySlug(&slug)
	}
	if threadErr != nil {
		if threadErr.Code == models.RowNotFound {
			utils.WriteEasyjson(w, http.StatusNotFound, threadErr)
			return
		}

		utils.WriteEasyjson(w, http.StatusInternalServerError, threadErr)
		return
	}

	utils.WriteEasyjson(w, http.StatusOK, thread)
}
