package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Id      int `gorm:"primaryKey;autoIncrement"`
	Name    string
	Age     int
	Email   string
	Balance float64
}

var Db *gorm.DB

func ConnectDb() {
	dsn := "host=localhost port=5432 user=postgres password=root dbname=ledger-app sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&User{})
	Db = db
	//defer db.Close()
}
