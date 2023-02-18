package main

import (
	"example.com/m/v2/database"
	"example.com/m/v2/router"
)

func main() {
	database.ConnectDb()
	router.Start()
}
