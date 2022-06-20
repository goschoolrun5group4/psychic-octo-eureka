package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	com "go-save-water/pkg/common"
	"io/ioutil"
	"net/http"

	"go-save-water/pkg/log"

	"golang.org/x/crypto/bcrypt"
)

func signup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type SignupUser struct {
			Username       string `json:"username"`
			FirstName      string `json:"firstName"`
			LastName       string `json:"lastName"`
			Password       string `json:"password,omitempty"`
			HashedPassword string `json:"hashedPassword,omitempty"`
			Email          string `json:"email"`
			Role           string `json:"role,omitempty"`
		}

		type LoginUser struct {
			UserID    int    `json:"userID"`
			Username  string `json:"username"`
			SessionID string `json:"sessionID"`
			ExpireDT  string `json:"expireDT"`
		}

		var (
			signupUser SignupUser
			loginUser  LoginUser
		)

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Server Error"))
			return
		}

		// convert JSON to object
		json.Unmarshal(reqBody, &signupUser)
		signupUser.Role = "user"

		bPassword, err := bcrypt.GenerateFromPassword([]byte(signupUser.Password), bcrypt.MinCost)
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Server Error"))
			return
		}
		signupUser.HashedPassword = string(bPassword)
		signupUser.Password = ""

		// Call User Create API
		jsonStr, err := json.Marshal(signupUser)
		if err != nil {
			fmt.Println(err)
			return
		}

		url := com.GetEnvVar("API_USER_ADDR") + "/user"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Internal Server Error"))
			return
		}

		if resp.StatusCode == http.StatusConflict {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("409 - Username Taken"))
			return
		}

		if resp.StatusCode == http.StatusUnprocessableEntity {
			log.Error.Println("SignUp not sending json to user create service.")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Internal Server Error"))
		}

		if resp.StatusCode == http.StatusInternalServerError {
			log.Error.Println("Error Processing from User create service")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Internal Server Error"))
		}

		// Process Session Data, delete old session if multiple sessions are detected.
		result := db.QueryRow("CALL spUserSessionCreate(?)", signupUser.Username)
		err = result.Scan(&loginUser.UserID, &loginUser.Username, &loginUser.SessionID, &loginUser.ExpireDT)
		// If user don't exist
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Server Error"))
			return
		}

		json.NewEncoder(w).Encode(loginUser)
	}
}

func login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type LoginUser struct {
			UserID    int    `json:"userID,omitempty"`
			Username  string `json:"username"`
			Password  string `json:"password,omitempty"`
			SessionID string `json:"sessionID,omitempty"`
			ExpireDT  string `json:"expireDT,omitempty"`
		}

		var (
			loginUser      LoginUser
			hashedPassword string
		)

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Server Error"))
			return
		}

		// convert JSON to object
		json.Unmarshal(reqBody, &loginUser)

		// Get user password from DB
		result := db.QueryRow("CALL spAuthenticationGet(?)", loginUser.Username)
		err = result.Scan(&hashedPassword)
		// If user don't exist
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Incorrect username or password."))
			return
		}

		// Compare passwords
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginUser.Password))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Incorrect username or password."))
			return
		}
		loginUser.Password = ""

		// Process Session Data, delete old session if multiple sessions are detected.
		result = db.QueryRow("CALL spUserSessionCreate(?)", loginUser.Username)
		err = result.Scan(&loginUser.UserID, &loginUser.Username, &loginUser.SessionID, &loginUser.ExpireDT)
		// If user don't exist
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Server Error"))
			return
		}

		json.NewEncoder(w).Encode(loginUser)
	}
}

func logout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type LogoutUser struct {
			UserID    int    `json:"userID"`
			SessionID string `json:"sessionID"`
		}

		var logoutUser LogoutUser

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Server Error"))
			return
		}

		// convert JSON to object
		json.Unmarshal(reqBody, &logoutUser)

		// Delete user session
		_, err = db.Query("call spUserSessionDelete(?, ?)", logoutUser.UserID, logoutUser.SessionID)
		if err != nil {
			log.Error.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Server Error"))
			return
		}

	}
}