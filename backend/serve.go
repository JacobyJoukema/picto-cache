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

	IMAGE_DIR = "./tmp"
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
		w.Write([]byte("500 - Failed to read file, try again later"))
		return
	}
	defer img.Close()

	// Read small part of file to ID content type
	buffer := make([]byte, 512)
	_, err = img.Read(buffer)
	if err != nil {
		logger.Error("failed to validate file type sending 400: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("400 - Failed to validate file type, ensure the file is correctly formatted as a jpeg (jpg) or png"))
		return
	}
	fileType := http.DetectContentType(buffer)

	// Reset the pointer location for writing later
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
	err = os.MkdirAll(fmt.Sprintf("%s/%v", IMAGE_DIR, uid), os.ModePerm)
	if err != nil {
		logger.Error("failed to establish image directory: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to read file, try again later"))
		return
	}

	// Prepare image meta for SQL storage
	imageData := Image{
		Uid:      int32(uid),
		Filename: imgHeader.Filename,
		Size:     int32(imgHeader.Size),
		Ref:      "", // placeholder reference for update after id is assigned to ensure unique filename
		Hidden:   hidden,
		Encoding: fileType,
	}

	// Insert image data and retrieve unique id
	imageData.Id, err = AddImageData(imageData)
	if err != nil {
		logger.Error("failed to add image meta: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to add image meta, try again later"))
		return
	}

	// Generate file reference string with unique file name in the format of IMAGE_DIR/UID/ID.ext
	imageData.Ref = fmt.Sprintf("%s/%v/%v%v", IMAGE_DIR, imageData.Uid, imageData.Id, filepath.Ext(imgHeader.Filename))

	// Update table with dynamic image reference
	// This is can be extended to support third party storage solutions
	err = UpdateImageData(imageData)
	if err != nil {
		logger.Error("failed to update metadata with image reference: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to update file referece in database, try again later"))

		DeleteImageData(imageData) // Clean DB for unsuccessful update

		return
	}

	// create file with reference string for writing
	fileRef, err := os.Create(imageData.Ref)
	if err != nil {
		logger.Error("failed to create file reference: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to create file reference, try again later"))

		DeleteImageData(imageData) // Clean DB for unsuccessful update
		return
	}

	// save the file at the reference
	_, err = io.Copy(fileRef, img)
	if err != nil {
		logger.Error("failed to save image: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to save file reference, try again later"))

		DeleteImageData(imageData) // Clean DB for unsuccessful update
		return
	}

	// marshal response in json
	js, err := json.Marshal(imageData)
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
