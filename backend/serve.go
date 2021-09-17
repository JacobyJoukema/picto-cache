package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/inflowml/logger"
)

const (
	PORT = ":8000"
)

type PingResp struct {
	Message string `json:"message"`
}

// Used for managing Image metadata tagged for json and sql serialization
type Image struct {
	Id       int32  `json:"id" sql:"id" typ:"SERIAL" opt:"PRIMARY KEY"`
	Uid      int32  `json:"uid" sql:"uid"`
	Filename string `json:"name" sql:"filename"`
	Ref      string `json:"ref" sql:"ref"`
	Size     int32  `json:"size" sql:"size"`
	Encoding string `json:"encoding" sql:"encoding"`
	Hidden   bool   `json:"hidden" sql:"hidden"`
}

// Used for managin User metadata tagged for json and sql serialization
type User struct {
	Uid       int32  `json:"uid" sql:"id" typ:"SERIAL" opt:"PRIMARY KEY"`
	Firstname string `json:"firstname" sql:"firstname"`
	Lastname  string `json:"lastname" sql:"lastname"`
}

func serve() error {

	router := mux.NewRouter()

	router.HandleFunc("/ping", ping)
	router.HandleFunc("/upload", upload)

	logger.Info("Initiating Server")
	return (http.ListenAndServe(PORT, router))
}

func ping(w http.ResponseWriter, req *http.Request) {
	logger.Debug("/ping request")

	resp := PingResp{Message: "pong"}
	js, err := json.Marshal(resp)
	if err != nil {
		logger.Error("failed to complete request: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func upload(w http.ResponseWriter, req *http.Request) {
	logger.Debug("/upload")

	_, meta, err := req.FormFile("image")
	if err != nil {
		logger.Error("failed to read file: %v", err)
		fmt.Fprintf(w, "upload failed")
		return
	}

	/*resp := Image{
		Filename: meta.Filename,
	}*/

	logger.Info("Filename: %v - Size: %v", meta.Filename, meta.Size)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "upload successful")
}
