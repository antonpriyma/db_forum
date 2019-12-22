package delivery

import (
	"github.com/AntonPriyma/db_forum/repository"
	"github.com/AntonPriyma/db_forum/utils"
	"net/http"
)

type ServiceHandlers struct {
	service *repository.DBService
}

func NewServiceHandlers(service *repository.DBService) *ServiceHandlers {
	return &ServiceHandlers{service: service}
}

func(h *ServiceHandlers) NewServiceHandlers(service *repository.DBService) *ServiceHandlers {
	return &ServiceHandlers{service: service}
}

func(h *ServiceHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetStatus()
	if err != nil {
		utils.WriteEasyjson(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteEasyjson(w, http.StatusOK, status)
}

func(h *ServiceHandlers) Clear(w http.ResponseWriter, r *http.Request) {
	err := h.service.Load()
	if err != nil {
		utils.WriteEasyjson(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
