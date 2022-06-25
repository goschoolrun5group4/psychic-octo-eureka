package server

import (
	"database/sql"
	"fmt"
	com "go-save-water/pkg/common"
	"go-save-water/pkg/log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var (
	user     = com.GetEnvVar("DB_USER")
	password = com.GetEnvVar("DB_PASSWORD")
	endpoint = com.GetEnvVar("DB_ENDPOINT")
	database = com.GetEnvVar("DB_DATABASE")
)

func Start() {
	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, endpoint, database)
	db, err := sql.Open("mysql", connectionString)
	fmt.Println("Server started")

	defer db.Close()

	if err != nil {
		log.Fatal.Fatalln("Error: Connection error")
	}
	if err = http.ListenAndServe(com.GetEnvVar("PORT"), handlers(db)); err != nil {
		log.Fatal.Fatalln("ListenAndServe: ", err)
	}
}

func handlers(db *sql.DB) http.Handler {
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()

	api.Handle("/usages/{accountNumber}", getUsages(db)).Methods("GET")
	api.Handle("/usage/{accountNumber}/{billDate}", getUsage(db)).Methods("GET")
	api.Handle("/usage", addUsage(db)).Methods("POST")
	api.Handle("/usage/{accountNumber}/{billDate}", updateUsage(db)).Methods("PUT")
	api.Handle("/usage/{accountNumber}/{billDate}", deleteUsage(db)).Methods("DELETE")

	return router
}
