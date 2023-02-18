package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	database "example.com/m/v2/database"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	database.ConnectDb()

	r := mux.NewRouter()

	r.HandleFunc("/create-user", createUserHandler).Methods("POST")
	r.HandleFunc("/add-balance", addBalanceHandler).Methods("POST")
	r.HandleFunc("/get-balance/{id}", getBalanceHandler).Methods("GET")

	http.ListenAndServe(":8080", r)
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user database.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := database.Db.Select("name", "age", "email").Create(&user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "User %s created successfully", user.Name)
}

func addBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var user database.User

	var requestPayload struct {
		Id      int     `json:"id"`
		Balance float64 `json:"balance"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := database.Db.First(&user, requestPayload.Id).Error; err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	newBalance := requestPayload.Balance + user.Balance
	if err := database.Db.Model(&user).Update("balance", newBalance).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "User %d balance updated successfully to %f", user.Id, newBalance)
}

func getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user database.User
	if err := database.Db.First(&user, id).Error; err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "User %d has %f balance", user.Id, user.Balance)
}
