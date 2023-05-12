package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	database "example.com/m/v2/database"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type ErrorMessage struct {
	Error string `json:"error"`
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user database.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if err := database.Db.Create(&user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	response := map[string]interface{}{
		"message": "User created successfully",
		"user":    user,
	}

	json.NewEncoder(w).Encode(response)
}

func AddBalance(w http.ResponseWriter, r *http.Request) {
	var user database.User

	var requestPayload struct {
		Id      int     `json:"id"`
		Balance float64 `json:"balance"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if requestPayload.Id == 0 || requestPayload.Balance == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)

		return
	}

	if err := database.Db.First(&user, requestPayload.Id).Error; err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "User not found"})

		return
	}

	newBalance := requestPayload.Balance + user.Balance
	if err := database.Db.Model(&user).Update("balance", newBalance).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	fmt.Fprintf(w, "User %d balance updated successfully to %f", user.Id, newBalance)
}

func GetBalance(w http.ResponseWriter, r *http.Request) {
	var user database.User
	email := r.FormValue("email")

	if err := database.Db.Where("email = ?", email).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)

		return
	}

	if err := database.Db.First(&user, id).Error; err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)

		return
	}

	fmt.Fprintf(w, "User %d has %f balance", user.Id, user.Balance)
}

func GetAllBalance(w http.ResponseWriter, r *http.Request) {
	var users []database.User

	if err := database.Db.Find(&users).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	type UserBalance struct {
		Id      int     `json:"id"`
		Balance float64 `json:"balance"`
	}

	var userBalances []UserBalance

	for _, user := range users {
		userBalances = append(userBalances, UserBalance{
			Id:      user.Id,
			Balance: user.Balance,
		})
	}

	response, err := json.Marshal(userBalances)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func TransferBalance(w http.ResponseWriter, r *http.Request) {
	var sender, receiver database.User

	var transferPayload struct {
		SenderID   int     `json:"sender_id"`
		ReceiverID int     `json:"receiver_id"`
		Amount     float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&transferPayload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if err := database.Db.First(&sender, transferPayload.SenderID).Error; err != nil {
		http.Error(w, "Sender not found", http.StatusBadRequest)

		return
	}

	if err := database.Db.First(&receiver, transferPayload.ReceiverID).Error; err != nil {
		http.Error(w, "Receiver not found", http.StatusBadRequest)

		return
	}

	str := fmt.Sprintf("%.10f", transferPayload.Amount)
	decimalPoints := len(str) - strings.Index(str, ".") - 1

	if decimalPoints > 2 {
		http.Error(w, "Transfer amount should be a multiple of 0.01", http.StatusBadRequest)

		return
	}

	if transferPayload.Amount <= 0 {
		http.Error(w, "Transfer amount should be greater than zero", http.StatusBadRequest)

		return
	}

	if sender.Balance < transferPayload.Amount {
		http.Error(w, "Insufficient balance", http.StatusBadRequest)

		return
	}

	tx := database.Db.Begin()

	oldSenderBalance := sender.Balance
	newSenderBalance := sender.Balance - transferPayload.Amount
	if err := tx.Model(&sender).Update("balance", newSenderBalance).Error; err != nil {
		tx.Rollback()

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	oldReceiverBalance := receiver.Balance
	newReceiverBalance := receiver.Balance + transferPayload.Amount
	if err := tx.Model(&receiver).Update("balance", newReceiverBalance).Error; err != nil {
		tx.Rollback()

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	transaction := database.Transaction{
		SenderId:           transferPayload.SenderID,
		ReceiverId:         transferPayload.ReceiverID,
		Type:               "transfer",
		Amount:             transferPayload.Amount,
		SenderOldBalance:   oldSenderBalance,
		SenderNewBalance:   newSenderBalance,
		ReceiverOldBalance: oldReceiverBalance,
		ReceiverNewBalance: newReceiverBalance,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	fmt.Fprintf(w, "Transfer of %f from user %d to user %d successful", transferPayload.Amount, transferPayload.SenderID, transferPayload.ReceiverID)
}

func Withdraw(w http.ResponseWriter, r *http.Request) {
	var user database.User

	var requestPayload struct {
		Id     int     `json:"id"`
		Amount float64 `json:"amount"`
	}

	email := r.FormValue("email")

	if err := database.Db.Where("email = ?", email).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if err := database.Db.First(&user, requestPayload.Id).Error; err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)

		return
	}

	if user.Balance < requestPayload.Amount {
		http.Error(w, "Insufficient balance", http.StatusBadRequest)

		return
	}

	newBalance := user.Balance - requestPayload.Amount
	if err := database.Db.Model(&user).Update("balance", newBalance).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	fmt.Fprintf(w, "User %d balance updated successfully to %f", user.Id, newBalance)
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	var user database.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	var existingUser database.User
	if err := database.Db.Where("email = ? AND deleted_at IS NULL", user.Email).First(&existingUser).Error; err == nil {
		http.Error(w, "Email already exists", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	if err := database.Db.Create(&user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "User created successfully",
		"user":    user,
	}

	json.NewEncoder(w).Encode(response)
}
