package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	Email     string `json:"email" sql:"email"`
}

// serve starts the http server and listens on port assigned above
func serve() error {

	router := mux.NewRouter()

	router.HandleFunc("/ping", ping)
	router.HandleFunc("/upload", upload)

	logger.Info("Initiating Server")
	return (http.ListenAndServe(PORT, router))
}

// ping responds to the url pattern /ping with a simple message to validate server
func ping(w http.ResponseWriter, req *http.Request) {
	logger.Debug("/ping request")

	resp := PingResp{
		Message: "pong",
	}
	js, err := json.Marshal(resp)
	if err != nil {
		logger.Error("failed to marshal json sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Something went wrong on our end"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// upload accepts multipart form-data with image metadata
// this function checks to ensure the image is of type jpg or png
func upload(w http.ResponseWriter, req *http.Request) {

	// Ensure request method is acceptable
	if req.Method != "POST" {
		logger.Error("%v request submitted to upload endpoint", req.Method)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - This endpoint only accepts post requests"))
		return
	}

	// attempt to retrieve file from form
	img, imgHeader, err := req.FormFile("img")
	if err != nil {
		logger.Error("failed to read file sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to read file, upload aborted"))
		return
	}
	defer img.Close()

	// Read small part of file to ID content type
	buffer := make([]byte, 512)
	_, err = img.Read(buffer)
	if err != nil {
		logger.Error("failed to validate file type sending 400: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("400 - Failed to validate file type, upload aborted, ensure the file is correctly formatted as a jpeg (jpg) or png"))
		return
	}
	fileType := http.DetectContentType(buffer)

	// Reset the location of reading to start of the file
	img.Seek(0, 0)

	logger.Info("Received For Upload Filename: %v - Size: %v - Type: %v", imgHeader.Filename, imgHeader.Size, fileType)
	// Validate Content-Type and image type
	contentType := req.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") || (fileType != "image/jpeg" && fileType != "image/png") {
		logger.Error("file type failure not accepted sending 400")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Failed to upload, please use multipart form data with an image of type jpeg (jpg) or png"))
		return
	}

	// TODO: replace with deserialization of jwt to auth user
	uid := 1

	// default to hidden unless explicitly false
	hidden := true
	if req.FormValue("hidden") == "false" {
		hidden = true
	}

	// ensure storage directory for the user exists
	err = os.MkdirAll(fmt.Sprintf("./tmp/%v", uid), os.ModePerm)
	if err != nil {
		logger.Error("failed to establish image directory: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to read file, upload aborted"))
		return
	}

	// Prepare image meta for SQL storage
	resp := Image{
		Uid:      int32(uid),
		Filename: imgHeader.Filename,
		Size:     int32(imgHeader.Size),
		Ref:      "", // placeholder reference for update after id is assigned to ensure unique filename
		Hidden:   hidden,
		Encoding: fileType,
	}

	// TODO: insert image meta into database to receive image id
	resp.Id = 1

	// Generate file reference string with unique file name in the format of ./tmp/UID/ID.ext
	refStr := fmt.Sprintf("./tmp/%v/%v%v", resp.Uid, resp.Id, filepath.Ext(imgHeader.Filename))

	// create file with reference string for writing
	fileRef, err := os.Create(refStr)
	if err != nil {
		logger.Error("failed to create file reference: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to create file reference, upload aborted"))
		return
	}

	// save the file at the reference
	_, err = io.Copy(fileRef, img)
	if err != nil {
		logger.Error("failed to save image: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to save file reference, upload aborted"))
		return
	}

	// TODO: update database with saved image
	resp.Ref = refStr

	// marshal response in json
	js, err := json.Marshal(resp)
	if err != nil {
		logger.Error("failed to marshal json sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Something went wrong on our end"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

	logger.Info("sucessfully uploaded image")
}
