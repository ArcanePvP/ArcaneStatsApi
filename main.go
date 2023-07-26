package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

type MojangUser struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type UserStats struct {
	Uuid       string
	Username   string
	Kills      int32
	Deaths     int32
	Coins      int32
	Killstreak int32
}

func HandleStats(wr http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		HandleGetStats(wr, req)
	case "POST":
		HandlePostStats(wr, req)
	case "DELETE":
		HandleDeleteStats(wr, req)
	}
}

func GetMojangUser(username string) *MojangUser {
	req, err := http.NewRequest("GET", "https://api.mojang.com/users/profiles/minecraft/"+username, nil)
	if err != nil {
		log.Fatal("Failed to get uuid from Mojang API: ", err)
		return &MojangUser{Name: username}
	}

	client := http.DefaultClient
	res, err := client.Do(req)

	if err != nil {
		log.Fatal("Failed to get uuid from Mojang API: ", err)
		return &MojangUser{Name: username}
	}

	defer res.Body.Close()

	mojangUser := MojangUser{}
	json.NewDecoder(res.Body).Decode(&mojangUser)

	return &mojangUser
}

func FullUuidFromTrimmed(trimmedId string) string {
	if strings.ContainsAny(trimmedId, "-") || trimmedId == "" {
		return trimmedId
	}
	return trimmedId[0:8] + "-" + trimmedId[8:12] + "-" + trimmedId[12:16] + "-" + trimmedId[16:20] + "-" + trimmedId[20:]
}

func HandleGetStats(wr http.ResponseWriter, req *http.Request) {
	wr.Header().Set("Content-Type", "application/json")
	wr.Header().Set("Access-Control-Allow-Origin", "*")
	//users/

	username := req.URL.Query().Get("username")

	if username == "" {
		username = ""
	}

	mojangUser := GetMojangUser(username)
	fmt.Println("Looking up stats of user " + FullUuidFromTrimmed(mojangUser.Id) + " (" + mojangUser.Name + ")")
	rows, err := db.Query("SELECT * FROM pvpstats WHERE uuid=?;", FullUuidFromTrimmed(mojangUser.Id))

	if err != nil {
		log.Fatal("Failed to read from database: ", err)
		return
	}

	defer rows.Close()

	type Response struct {
		ErrorMessage string `json:"error_message"`
	}

	if !rows.Next() {
		wr.WriteHeader(http.StatusNotFound)
		json.NewEncoder(wr).Encode(Response{"No stats available for this player."})
		return
	}

	var uuid string
	var kills,
		deaths,
		coins,
		killstreak int32

	rows.Scan(&uuid, &kills, &deaths, &coins, &killstreak)

	json.NewEncoder(wr).Encode(UserStats{
		Uuid:       uuid,
		Username:   mojangUser.Name,
		Kills:      kills,
		Deaths:     deaths,
		Coins:      coins,
		Killstreak: killstreak,
	})
}

func HandlePostStats(wr http.ResponseWriter, req *http.Request) {
}

func HandleDeleteStats(wr http.ResponseWriter, req *http.Request) {
}

func ConnectDb() *sql.DB {
	db, err := sql.Open("mysql", "u26609_dWb5joQCUq:w2r11gGx4nEHOPbj1aoBsyML@tcp(mahimahi.bloom.host:3306)/s26609_stats")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
	//fmt.Println("Successfully connected to database!")
}

func main() {
	db = ConnectDb()
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/stats", HandleStats)

	fmt.Println("Starting HTTP server on port 8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println("Failed to start HTTP server ", err)
	}
}
