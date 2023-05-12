package router

import (
	"net/http"

	"github.com/gorilla/mux"

	handlers "example.com/m/v2/handlers"
)

func Start() {
	router := mux.NewRouter()

	router.HandleFunc("/create-user", handlers.CreateUser).Methods("POST")
	router.HandleFunc("/add-balance", handlers.AddBalance).Methods("POST")
	router.HandleFunc("/get-balance/{id}", handlers.GetBalance).Methods("GET")
	router.HandleFunc("/get-all-balance", handlers.GetAllBalance).Methods("GET")
	router.HandleFunc("/transfer-balance", handlers.TransferBalance).Methods("POST")
	router.HandleFunc("/withdraw", handlers.Withdraw).Methods("POST")
	router.HandleFunc("/signup", handlers.SignUp).Methods("POST")

	http.ListenAndServe(":8080", router)
}
