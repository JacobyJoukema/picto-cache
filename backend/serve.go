package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/inflowml/logger"
	"golang.org/x/crypto/bcrypt"
)

const (
	PORT = ":8000"

	IMAGE_DIR = "image"
	REF_URL   = "localhost:8000" // Default if REF_URL env variable is not defined
)

// Test server secret for non-production deployment
// Use SIGNING_KEY environment variable for production or appropriately stored key
var SIGNING_KEY = []byte("hirejacobyjoukema")

type PingResp struct {
	Message string `json:"message"`
}

// Used for managing Image metadata tagged for json and sql serialization
type Image struct {
	Id        int32  `json:"id" sql:"id" typ:"SERIAL" opt:"PRIMARY KEY"`
	Uid       int32  `json:"uid" sql:"uid"`
	Title     string `json:"title" sql:"title"`
	Ref       string `json:"ref" sql:"ref"`
	Size      int32  `json:"size" sql:"size"`
	Encoding  string `json:"encoding" sql:"encoding"`
	Shareable bool   `json:"shareable" sql:"shareable"`
	// UploadDate Expansion opportunity
}

type QueryResp struct {
	Page         int     `json:"page"`
	PageSize     int     `json:"pageSize"`
	TotalResults int     `json:"totalResults"`
	ImageMeta    []Image `json:"imageMeta"`
}

// ImageParams are mutable parameters that can be defined by users
// these can be expanded to allow for more user defined features like tags, ratings, likes, prices
type ImageParams struct {
	Title     string `json:"title"`
	Shareable bool   `json:"shareable"`
	// Rating Expansion opportunity
	// Tags     []byte `json:"tags" sql:"tags"` // Expansion opportunity, tagging images
}

// Used for managing User metadata tagged for json and sql serialization
// Separated from UserPassword as this struct is front facing
type User struct {
	Uid       int32  `json:"uid" sql:"id" typ:"SERIAL" opt:"PRIMARY KEY"`
	Firstname string `json:"firstname" sql:"firstname"`
	Lastname  string `json:"lastname" sql:"lastname"`
	Email     string `json:"email" sql:"email"`
}

// Used for managing User Passwords hashed passwords
// Separated from User table as this is not for public vision
type UserPassword struct {
	Uid        int32  `sql:"id" opt:"PRIMARY KEY"` // Corresponds to User Uid
	HashedPass string `sql:"hashed_pass"`
}

type TokenResp struct {
	Name       string `json:"name"`
	Value      string `json:"token"`
	Expiration string `json:"expiration"`
}

type JWTClaims struct {
	Email string
	Uid   int
	jwt.StandardClaims
}

// serve starts the http server and listens on port assigned above
func serve() error {

	router := mux.NewRouter()

	// Basic service endpoints
	router.HandleFunc("/ping", ping).Methods("GET", "OPTIONS")
	router.HandleFunc("/register", register).Methods("POST", "OPTIONS")
	router.HandleFunc("/auth", auth).Methods("GET", "OPTIONS")

	// Basic image creation endpoint
	router.HandleFunc("/image", addImage).Methods("POST", "OPTIONS")

	// Image data endpoints
	router.HandleFunc("/image/{uid:[0-9]+}/{fileId}", getImage).Methods("GET", "OPTIONS")
	router.HandleFunc("/image/{uid:[0-9]+}/{fileId}", delImage).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/image/{uid:[0-9]+}/{fileId}", updateImage).Methods("PUT", "OPTIONS")

	// Image meta query methods
	router.HandleFunc("/image/meta?", imageMetaRequest).Queries(
		"page", "{page:[0-9]+}",
		"id", "{id:[0-9]+}",
		"uid", "{uid:[0-9]+}",
		"title", "{title}",
		"encoding", "{encoding}",
		"shareable", "{shareable)").Methods("GET")
	router.HandleFunc("/image/meta", imageMetaRequest).Methods("GET")

	http.Handle("/", router)

	logger.Info("Initiating HTTP Server")
	//return http.ListenAndServeTLS(":5050", "./keys/server.crt", "./keys/server.key", router)
	return (http.ListenAndServe(PORT, router))
}

// ping responds to the url pattern /ping with a simple message to validate server
func ping(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

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

func register(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

	// Ensure request is multipart/form-data
	contentType := req.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		logger.Error("bad request content type sending 400")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Content-Type header incorrect ensure that body is multipart/form-data"))
		return
	}

	// Define the user struct out of provided details
	user := User{
		Email:     req.FormValue("email"),
		Firstname: req.FormValue("firstname"),
		Lastname:  req.FormValue("lastname"),
	}
	password := req.FormValue("password")

	// Validate all required fields are completed
	if len(user.Email) == 0 || len(user.Firstname) == 0 || len(user.Lastname) == 0 || len(password) == 0 {
		logger.Error("Bad request, required fields are empty returning 400")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Required fields are empty, correct request and try again"))
		return
	}

	// Ensure email isn't already registered
	emailUnique, err := UniqueEmail(user.Email)
	if err != nil {
		logger.Error("Unable to validate email sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to register account try again later"))
		return
	}

	// Return failed request for pre-registered email
	if !emailUnique {
		logger.Error("Email already exists sending 400")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - That email already exists, login or register with a different email"))
		return
	}

	// Add user to database
	user.Uid, err = AddUserData(user)
	if err != nil {
		logger.Error("Unable to add account to database sending 500")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to register account try again later"))
		return
	}

	// Attempt to hash password for storage
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Failed to hash password cleaning user and sending 500: %v", err)
		w.WriteHeader((http.StatusInternalServerError))
		w.Write([]byte("500 - Unable to hash password try again later"))
		DeleteUserData(user)
		return
	}

	pass := UserPassword{
		Uid:        user.Uid,
		HashedPass: string(hashedPass),
	}

	// Add hashed password to password table
	uid, err := AddUserPass(pass)
	if err != nil {
		logger.Error("Failed to store hashed password cleaning user and sending 500: %v", err)
		w.WriteHeader((http.StatusInternalServerError))
		w.Write([]byte("500 - Unable to store hash password try again later"))
		DeleteUserData(user)
		return
	}

	logger.Info("UID: %v - UID PASS: %v", user.Uid, uid)

	// Generate and set JWT
	token, exp, err := generateJWT(int(user.Uid), user.Email)
	if err != nil {
		logger.Error("Failed to generate jwt, sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized, unable to generate valid token"))
		return
	}

	// Set JWT Cookie with the name token
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   token,
		Expires: time.Unix(exp, 0),
	})

	// Prepare to marshal into json
	tokenResp := TokenResp{
		Name:       "token",
		Value:      token,
		Expiration: time.Unix(exp, 0).String(),
	}

	resp, err := json.Marshal(tokenResp)
	if err != nil {
		logger.Error("failed to marshal token, sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Unable to marshal token, try again later"))
		return
	}

	w.Write(resp)
	return
}

func auth(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

	// Retrieve basic auth credentials
	email, password, _ := req.BasicAuth()

	hashedPass, user, err := GetHashedPass(email)
	if err != nil {
		logger.Error("Unable to retrieve hashed password, sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized, unable to verify this login attempt"))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(password))
	if err != nil {
		logger.Error("Password mismatch, sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized, invalid login"))
		return
	}

	logger.Info("Successfull login for user: %v", email)

	// Generate and set JWT
	token, exp, err := generateJWT(int(user.Uid), user.Email)
	if err != nil {
		logger.Error("Failed to generate jwt, sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized, unable to generate valid token"))
		return
	}

	// Set JWT Cookie with the name token
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   token,
		Expires: time.Unix(exp, 0),
	})

	// Prepare to marshal into json
	tokenResp := TokenResp{
		Name:       "token",
		Value:      token,
		Expiration: time.Unix(exp, 0).String(),
	}

	resp, err := json.Marshal(tokenResp)
	if err != nil {
		logger.Error("failed to marshal token, sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Unable to marshal token, try again later"))
		return
	}

	w.Write(resp)
	return
}

func generateJWT(uid int, email string) (string, int64, error) {

	// Set expiration to 30 minutes from login
	exp := time.Now().Add(time.Minute * 30).Unix()

	claims := &JWTClaims{
		Email: email,
		Uid:   uid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp,
		},
	}
	signingKey := getSigningKey()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenStr, err := token.SignedString(signingKey)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign jwt: %v", err)
	}

	return tokenStr, exp, err
}

// getSigningKey retrievs the secret key from the SIGNING_KEY environent variable
// this function can be replaced with other methods for retrieving keys for example if
// they are stored on disk as a PEM or similar file
func getSigningKey() []byte {
	// Get signing key
	signingKey := []byte(os.Getenv("SIGNING_KEY"))
	if len(signingKey) == 0 {
		signingKey = SIGNING_KEY
	}

	return signingKey
}

// authRequest accepts the http request and parses the attached jwt token
// and returns the JWTClaims for the assigned jwt which is stored
// in a cookie. Users also have the opportunity to use the token as bearer token
func authRequest(req *http.Request) (JWTClaims, error) {

	// init tokenStr
	tokenStr := ""

	// attempt to retrieve from cookie, if not assign the value of the authorization header
	cookie, err := req.Cookie("token")
	if err != nil {
		tokenStr = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	} else {
		tokenStr = cookie.Value
	}

	logger.Info(tokenStr)

	claims := &JWTClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return getSigningKey(), nil
	})
	if err != nil || !token.Valid {
		return JWTClaims{}, fmt.Errorf("failed to parse jwt/invalid token, unauthorized")
	}

	return *claims, nil
}

// getImage returns the image defined in the url parameters if the user is authorized to view it
func getImage(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

	logger.Info("hit getImage end")
	// Authorize request
	claims, err := authRequest(req)
	if err != nil {
		logger.Error("Unauthorized request to upload sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized request, ensure you sign in and obtain the jwt auth token"))
		return
	}

	vars := mux.Vars(req)

	// validate url parameters and retrieve imageMeta
	// returns a 404 if data cannot be found in the db otherwise assumes bad request
	imageMeta, err := validateVars(vars)
	if err != nil {
		if err != nil {
			logger.Error("Failed to validate vars sending 400: %v", err)
			if strings.Contains(err.Error(), "404 - Not found") {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("404 - Not found, no image with that information available"))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("400 - Bad request unable to parse url parameters"))
			return
		}
	}

	// Ensure user has access permissions
	if claims.Uid != int(imageMeta.Uid) {
		logger.Error("unauthorized user attempting to delete image")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized, this file is private and you do not have access"))
		return
	}

	// prepare file for sending
	fileBytes, err := ioutil.ReadFile(fmt.Sprintf("./%s/%s/%s", IMAGE_DIR, vars["uid"], vars["fileId"]))
	if err != nil {
		logger.Error("Failed to retrieve file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to retrieve file, try again later"))
	}

	w.Header().Set("Content-Type", imageMeta.Encoding)
	w.Write(fileBytes)
	return
}

// addImage accepts multipart form-data with image metadata
// this function checks to ensure the image is of type jpg or png
func addImage(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

	claims, err := authRequest(req)
	if err != nil {
		logger.Error("Unauthorized request to upload sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized request, ensure you sign in and obtain the jwt auth token"))
		return
	}

	// attempt to retrieve file from form
	img, imgHeader, err := req.FormFile("image")
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

	// Read enough of file to determine type
	fileType := http.DetectContentType(buffer)

	// Reset the pointer location for writing later
	img.Seek(0, 0)

	// Validate Content-Type and image type
	contentType := req.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") || (fileType != "image/jpeg" && fileType != "image/png") {
		logger.Error("file type failure not accepted sending 400: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Failed to upload, please use multipart form data with an image of type jpeg (jpg) or png"))
		return
	}

	// Generate file extension based on data type
	fileExt := strings.Split(fileType, "/")[1]

	uid := claims.Uid

	// default to not shareable unless explicitly false
	shareable := false
	if req.FormValue("shareable") == "true" {
		shareable = true
	}

	// ensure storage directory for the user exists
	err = os.MkdirAll(fmt.Sprintf("./%s/%v", IMAGE_DIR, uid), os.ModePerm)
	if err != nil {
		logger.Error("failed to establish image directory: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to read file, try again later"))
		return
	}

	// Determine if filename exists
	title := req.FormValue("title")
	if len(title) == 0 {
		title = imgHeader.Filename
	}

	// Manually assign extension even if one is already there
	title = fmt.Sprintf("%s.%s", strings.Split(title, ".")[0], fileExt)

	// Prepare image meta for SQL storage
	imageData := Image{
		Uid:       int32(uid),
		Title:     title,
		Size:      int32(imgHeader.Size),
		Ref:       "", // placeholder reference for update after id is assigned to ensure unique filename
		Shareable: shareable,
		Encoding:  fileType,
	}

	// Insert image data and retrieve unique id
	imageData.Id, err = AddImageData(imageData)
	if err != nil {
		logger.Error("failed to add image meta: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to add image meta, try again later"))
		return
	}

	// Get REF_URL
	refUrl := os.Getenv("REF_URL")
	if len(refUrl) == 0 {
		refUrl = REF_URL
	}

	// Generate file reference string with unique file name in the format of IMAGE_DIR/UID/ID.ext
	imageData.Ref = fmt.Sprintf("%s/%s/%v/%v.%v", refUrl, IMAGE_DIR, imageData.Uid, imageData.Id, fileExt)

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

	// Generate local file reference string
	fileRefStr := fmt.Sprintf("./%s/%v/%v.%v", IMAGE_DIR, imageData.Uid, imageData.Id, fileExt)

	// create file with reference string for writing
	fileRef, err := os.Create(fileRefStr)
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
	logger.Info("Successfully uploaded (Title: %v - Size: %v - Type: %v)", title, imgHeader.Size, fileType)
}

// delImage accepts multipart form-data with image metadata and deletes the appropriate
// image given the requesting person has the authorization to do so
func delImage(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

	logger.Info("hit delImage end")

	// Authenticate user
	claims, err := authRequest(req)
	if err != nil {
		logger.Error("Unauthorized request to upload sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized request, ensure you sign in and obtain the jwt auth token"))
		return
	}

	vars := mux.Vars(req)
	// validate url parameters and retrieve imageMeta
	imageMeta, err := validateVars(vars)
	if err != nil {
		logger.Error("Failed to validate vars sending 400: %v", err)
		if strings.Contains(err.Error(), "404 - Not found") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 - Not found, no image with that information available"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Bad request unable to parse url parameters"))
		return
	}

	// Ensure there is no uid miss match
	uidVal, err := strconv.Atoi(vars["uid"])
	if uidVal != int(imageMeta.Uid) {
		logger.Error("uid miss match when attempting to delete image sending 400")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Uid mismatch ensure you are using the correct image reference"))
		return
	}

	// Ensure user has access permissions
	if claims.Uid != int(imageMeta.Uid) {
		logger.Error("unauthorized user attempting to delete image")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized, you do not have permissions to modify this image"))
		return
	}

	// Delete meta from database
	err = DeleteImageData(imageMeta)
	if err != nil {
		logger.Error("failed to delete image from database sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Unable to delete image from database, try again later"))
		return
	}

	// Delete file from storage
	fileRef := fmt.Sprintf("./%s/%s/%s", IMAGE_DIR, vars["uid"], vars["fileId"])
	err = os.Remove(fileRef)
	// Orphaned file is ok to leave as database entry is already deleted
	// Automated data integrity checks or manual removal is recommended
	// This will look like a successfull deletion from the users perspective
	if err != nil {
		logger.Error("failed to delete image data, clean orphaned files via automated data integrity check: %v", err)
	} else {
		logger.Info("Successfully deleted image: %v", imageMeta.Id)
	}

	return
}

// getImage accepts multipart form-data with image metadata and deletes the appropriate
// image given the requesting person has the authorization to do so
func imageMetaRequest(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

	// Authenticate user
	claims, err := authRequest(req)
	if err != nil {
		logger.Error("Unauthorized request to upload sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized request, ensure you sign in and obtain the jwt auth token"))
		return
	}

	params := req.URL.Query()

	resp, err := ImageMetaQuery(claims.Uid, params)
	if err != nil {
		logger.Error("failed to retrieve image metadata: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to complete query, try again later"))
		return
	}

	// marshal data into json to prep the query response
	js, err := json.Marshal(resp)
	if err != nil {
		logger.Error("Failed to marshal image meta sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - failed to marshal response, try again later"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
	return
}

// getImage accepts multipart form-data with image metadata and deletes the appropriate
// image given the requesting person has the authorization to do so
func updateImage(w http.ResponseWriter, req *http.Request) {

	// Manage Cors
	setCors(&w)
	if req.Method == "OPTIONS" {
		return
	}

	// Authenticate user
	claims, err := authRequest(req)
	if err != nil {
		logger.Error("Unauthorized request to upload sending 401: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized request, ensure you sign in and obtain the jwt auth token"))
		return
	}

	vars := mux.Vars(req)
	// validate url parameters and retrieve imageMeta
	imageMeta, err := validateVars(vars)
	if err != nil {
		if strings.Contains(err.Error(), "404 - Not found") {
			logger.Error("image data does not exist sending 404: %v", err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 - Not found, no image with that information available"))
			return
		}
		logger.Error("Failed to validate vars sending 400: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Bad request unable to parse url parameters"))
		return
	}

	// Ensure there is no uid miss match
	uidVal, err := strconv.Atoi(vars["uid"])
	if uidVal != int(imageMeta.Uid) {
		logger.Error("uid miss match when attempting to modify image sending 400")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Uid mismatch ensure you are using the correct image reference"))
		return
	}

	// Ensure user has access permissions
	if claims.Uid != int(imageMeta.Uid) {
		logger.Error("unauthorized user attempting to modify image")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized, you do not have permissions to modify this image"))
		return
	}

	// decode json message into ImageParams struct
	// string map must be used to account for empty values
	var newParams map[string]string
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&newParams)
	if err != nil {
		logger.Error("failed to demarshal json body sending 400: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - unable to parse json, check your request"))
		return
	}

	// if request specified a new title that is at least one character update meta
	if title, ok := newParams["title"]; ok && len(title) > 0 {
		fileExt := strings.Split(imageMeta.Encoding, "/")[1]

		// Manually assign extension even if one is already there
		imageMeta.Title = fmt.Sprintf("%s.%s", strings.Split(title, ".")[0], fileExt)
	}

	// if request specified a new shareable value that is valid update meta
	if shareable, ok := newParams["shareable"]; ok {
		if shareable == "true" {
			imageMeta.Shareable = true
		} else if shareable == "false" {
			imageMeta.Shareable = false
		}
	}

	err = UpdateImageData(imageMeta)
	if err != nil {
		logger.Error("failed to update database with new meta sending 500: %v")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Failed to update database, try again later"))
		return
	}

	// marshal data into json to prep the query response
	js, err := json.Marshal(imageMeta)
	if err != nil {
		logger.Error("Failed to marshal image meta sending 500: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - failed to marshal response, try again later"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
	logger.Info("Successfully returned image meta request for UID: %v", claims.Uid)

	return

}

func validateVars(vars map[string]string) (Image, error) {

	// Validate completeness of request
	if len(vars["uid"]) == 0 || len(vars["fileId"]) == 0 {
		return Image{}, fmt.Errorf("incomplete image request, null parameters")
	}

	// Parse file id and convert to int
	id, err := strconv.Atoi(strings.TrimSuffix(vars["fileId"], filepath.Ext(vars["fileId"])))
	if err != nil {
		return Image{}, fmt.Errorf("unable to parse file id: %v", err)
	}

	// Retreive image meta
	imageMeta, err := GetImageMeta(int32(id))
	if err != nil {
		return Image{}, fmt.Errorf("unable to retreive image meta from database: %v", err)
	}

	return imageMeta, nil
}

func setCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}
