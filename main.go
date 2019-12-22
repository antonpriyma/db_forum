package main

import (
	"log"
	"net/http"

	"github.com/AntonPriyma/db_forum/delivery"
	"github.com/AntonPriyma/db_forum/repository"
	"github.com/gorilla/mux"
)

// Handler структура хэндлера запросов
type Handler struct {
	Router *mux.Router
}

func main() {

	//connectError := models.ConnetctDB("docker", "docker", "localhost", "docker")
	dbService := repository.NewDBService()
	connectError := repository.ConnetctDB(dbService, "docker", "docker", "localhost", "docker")
	if connectError != nil {
		log.Fatalf("cant open database connection: %s", connectError.Message)
	}
	forumRepo := repository.NewForumRepositoryImpl(repository.GetDB())
	usersRepo := repository.NewUsersRepositoryImpl(repository.GetDB(),forumRepo)
	threadsRepo := repository.NewThreadDBRepositoryImpl(repository.GetDB())
	postsRepo := repository.NewPostDBRepositoryImpl(usersRepo,threadsRepo,repository.GetDB())


	users := delivery.NewUsersHandlers(usersRepo)
	threads := delivery.NewThreadHandlers(threadsRepo)
	posts := delivery.NewPostHandlers(postsRepo, usersRepo)
	forums := delivery.NewForumHandlers(forumRepo, usersRepo)
	service := delivery.NewServiceHandlers(dbService)
	r := mux.NewRouter().PathPrefix("/api").Subrouter()
	r.HandleFunc("/user/{nickname}/profile", users.GetUser).Methods("GET")
	r.HandleFunc("/user/{nickname}/create", users.CreateUser).Methods("POST")
	r.HandleFunc("/user/{nickname}/profile", users.UpdateUser).Methods("POST")

	r.HandleFunc("/forum/create", forums.CreateForum).Methods("POST")
	r.HandleFunc("/forum/{slug}/details", forums.GetForum).Methods("GET")
	r.HandleFunc("/forum/{slug}/create", threads.CreateThread).Methods("POST")
	r.HandleFunc("/forum/{slug}/threads", threads.GetThreadsByForum).Methods("GET")
	r.HandleFunc("/forum/{slug}/users", forums.GetForumUsers).Methods("GET")

	r.HandleFunc("/thread/{slug_or_id}/create", posts.CreatePosts).Methods("POST")
	r.HandleFunc("/thread/{slug_or_id}/vote", threads.Vote).Methods("POST")
	r.HandleFunc("/thread/{slug_or_id}/details", threads.GetThread).Methods("GET")
	r.HandleFunc("/thread/{slug_or_id}/posts", posts.GetPosts).Methods("GET")
	r.HandleFunc("/thread/{slug_or_id}/details", threads.UpdateThread).Methods("POST")

	r.HandleFunc("/post/{id:[0-9]+}/details", posts.GetPost).Methods("GET")
	r.HandleFunc("/post/{id:[0-9]+}/details", posts.UpdatePost).Methods("POST")

	r.HandleFunc("/service/status", service.GetStatus).Methods("GET")
	r.HandleFunc("/service/clear", service.Clear).Methods("POST")

	h := Handler{
		Router: r,
	}

	port := "5000"
	log.Printf("MainService successfully started at port %s", port)
	err := http.ListenAndServe(":"+port, h.Router)
	if err != nil {
		log.Fatalf("cant start main server. err: %s", err.Error())
	}
}
