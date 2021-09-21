package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type RouteTest struct {
	Route    string
	Func     func(w http.ResponseWriter, req *http.Request)
	Method   []string
	Expected []int
}

var testUser = User{
	Firstname: "Jacoby",
	Lastname:  "Joukema",
	Email:     "user@mail.com",
}
var userPass = "pass"

// TestRouting evaluates a number of endpoints without authentication and ensures the correct response headers
// This is a catch all for routing detailed tests of endpoint edge cases are completed in
// the appropriate test function.
func TestRouting(t *testing.T) {
	router := configureRoutes()

	// Setup testing parameters
	routeTests := []RouteTest{
		{
			Route:    "/",
			Func:     home,
			Method:   []string{"GET", "OPTIONS", "POST", "PUT", "DELETE"},
			Expected: []int{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK},
		}, {
			Route:    "/ping",
			Func:     ping,
			Method:   []string{"GET", "OPTIONS", "POST", "PUT", "DELETE"},
			Expected: []int{http.StatusOK, http.StatusOK, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed},
		}, {
			Route:    "/register",
			Func:     register,
			Method:   []string{"GET", "OPTIONS", "POST", "PUT", "DELETE"},
			Expected: []int{http.StatusMethodNotAllowed, http.StatusOK, http.StatusBadRequest, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed},
		}, {
			Route:    "/auth",
			Func:     auth,
			Method:   []string{"GET", "OPTIONS", "POST", "PUT", "DELETE"},
			Expected: []int{http.StatusUnauthorized, http.StatusOK, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed},
		}, {
			Route:    "/image",
			Func:     addImage,
			Method:   []string{"GET", "OPTIONS", "POST", "PUT", "DELETE"},
			Expected: []int{http.StatusMethodNotAllowed, http.StatusOK, http.StatusUnauthorized, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed},
		}, {
			Route:    "/image/1/1.png",
			Func:     getImage,
			Method:   []string{"GET", "OPTIONS", "POST", "PUT", "DELETE"},
			Expected: []int{http.StatusUnauthorized, http.StatusOK, http.StatusMethodNotAllowed, http.StatusUnauthorized, http.StatusUnauthorized},
		}, {
			Route:    "/image/meta",
			Func:     imageMetaRequest,
			Method:   []string{"GET", "OPTIONS", "POST", "PUT", "DELETE"},
			Expected: []int{http.StatusUnauthorized, http.StatusOK, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed, http.StatusMethodNotAllowed},
		},
	}

	// Iterate through requests and evaluate
	for _, routeTest := range routeTests {
		// Iterate through methods and check responses
		for i, method := range routeTest.Method {
			req, err := http.NewRequest(method, routeTest.Route, nil)
			if err != nil {
				t.Fatal(fmt.Errorf("failed to form request for %s method %s: %v", routeTest.Route, method, err))
			}
			// Set up request callers
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Validate status codes
			if status := rr.Code; status != routeTest.Expected[i] {
				t.Errorf("handler returned wrong code for %s method %s: got %v want %v", routeTest.Route, method, status, routeTest.Expected[i])
			}
		}
	}
}

// TestPingHandler ensures correct response for a valid /ping request
func TestPingHandler(t *testing.T) {

	router := configureRoutes()

	// Request recorder init
	rr := httptest.NewRecorder()

	// Define /ping request
	req, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Submit request
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusOK)
	}

	expected := PingResp{
		Message: "pong",
	}
	resp := PingResp{}
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, resp) {
		t.Errorf("handler returned wrong response: got %v want %v", resp, expected)
	}
}

// TestRegister sends valid and invalid multipart form-data to the /register endpoint
// This test evaluates the response status and response body
func TestRegister(t *testing.T) {

	// Configure http message
	router := configureRoutes()

	// Request recorder init
	rr := httptest.NewRecorder()

	// Configure http request without any data
	req, err := http.NewRequest("POST", "/register", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Add authentication

	// Send invalid request
	router.ServeHTTP(rr, req)

	// Compare status codes expect bad request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusBadRequest)
	}

	// Generate incomplete multipart form data
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)

	err = writer.WriteField("firstname", testUser.Firstname)
	if err != nil {
		t.Errorf("failed to create form field: %v", err)
	}
	err = writer.WriteField("lastname", testUser.Lastname)
	if err != nil {
		t.Errorf("failed to create form field: %v", err)
	}
	err = writer.WriteField("password", userPass)
	if err != nil {
		t.Errorf("failed to create form field: %v", err)
	}

	// prepare incomplete request
	req, err = http.NewRequest("POST", "/register", bytes.NewReader(form.Bytes()))
	if err != nil {
		t.Errorf("failed to send POST request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Refresh recorder
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	// Compare status codes expect bad request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusBadRequest)
	}

	// Complete request body and retry
	err = writer.WriteField("email", testUser.Email)
	if err != nil {
		t.Errorf("failed to create form field: %v", err)
	}

	writer.Close()

	// Send complete request
	req, err = http.NewRequest("POST", "/register", bytes.NewReader(form.Bytes()))
	if err != nil {
		t.Errorf("failed to send POST request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Refresh recorder
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Compare status codes expect bad request
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusOK)
	}

	// Submit additional request with same user
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	// Compare status codes expect bad request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusBadRequest)
	}

	err = deleteTestUser()

}

// TestAuth tests the /auth endpoint for a valid and an invalid credential
func TestAuth(t *testing.T) {

	// Create testUser
	_, err := createTestUser()
	if err != nil {
		t.Errorf("failed to create test user: %v", err)
	}

	// Configure http message
	router := configureRoutes()

	// Request recorder init
	rr := httptest.NewRecorder()

	// Configure http request
	req, err := http.NewRequest("GET", "/auth", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Set valid auth header
	auth := fmt.Sprintf("%s:%s", testUser.Email, userPass)
	auth = fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth)))
	req.Header.Add("Authorization", auth)

	router.ServeHTTP(rr, req)

	// Compare status codes expect bad request
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusOK)
	}

	// Configure http message
	router = configureRoutes()

	// Request recorder init
	rr = httptest.NewRecorder()

	// Configure http request
	req, err = http.NewRequest("GET", "/auth", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set invalid auth header
	auth = fmt.Sprintf("%s:%s", testUser.Email, "badpass")
	auth = fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth)))
	req.Header.Add("Authorization", auth)

	router.ServeHTTP(rr, req)

	// Compare status codes expect bad request
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusUnauthorized)
	}

	// Cleanup database
	err = deleteTestUser()
	if err != nil {
		t.Errorf("failed to delete test user: %v", err)
	}
}

// TestUploadImage attempts to upload a file via the /image post request
// This test requires an image name test.png in the ./test/test.png directory
func TestUploadImage(t *testing.T) {
	token, uid, err := getTestToken()
	if err != nil {
		t.Errorf("failed to generate test user jwt token: %v", err)
	}

	// Generate incomplete multipart form data
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)

	err = writer.WriteField("shareable", "true")
	if err != nil {
		t.Errorf("failed to create form field: %v", err)
	}
	err = writer.WriteField("title", "image.png")
	if err != nil {
		t.Errorf("failed to create form field: %v", err)
	}

	file, err := os.Open("./test/test.png")
	if err != nil {
		t.Errorf("failed to open ./test/test.png: %v", err)
	}
	part, _ := writer.CreateFormFile("image", "./test/test.png")
	io.Copy(part, file)
	writer.Close()

	req, err := http.NewRequest("POST", "/image", bytes.NewReader(form.Bytes()))
	if err != nil {
		t.Errorf("failed to generate request with form data: %v", err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	// Configure http message
	router := configureRoutes()

	// Request recorder init
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Compare status codes expect bad request
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong code: got %v want %v", status, http.StatusOK)
	}

	// Read body to clean
	imageMeta := Image{}
	err = json.Unmarshal(rr.Body.Bytes(), &imageMeta)
	if err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	// clean image meta from database
	err = DeleteImageData(imageMeta)
	if err != nil {
		t.Errorf("failed to delete image data meta: %v", err)
	}

	err = os.RemoveAll(fmt.Sprintf("./%s/%v", IMAGE_DIR, uid))
	if err != nil {
		t.Errorf("failed to delete image data: %v", err)
	}

	// Clean file upload

	/*err = deleteTestUser()
	if err != nil {
		t.Errorf("failed to delete test user: %v", err)
	}*/
}

// TestGetImage attempts to retrieve an image from the database
func TestGetImage(t *testing.T) {

}

// getTestToken generates a token after creating a test user
// must call delete test user at the end of the request
func getTestToken() (string, int, error) {
	uid, err := createTestUser()
	if err != nil {
		return "", 0, fmt.Errorf("failed to create test user: %v", err)
	}
	token, _, err := generateJWT(uid, testUser.Email)
	return token, uid, err
}

// createTestUser is a helper function that populates the database with the default test user defined above
func createTestUser() (int, error) {

	uid, err := AddUserData(testUser)
	if err != nil {
		return 0, fmt.Errorf("unable to add test user: %v", err)
	}

	user := testUser
	user.Uid = uid

	// Attempt to hash password for storage
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(userPass), bcrypt.DefaultCost)
	if err != nil {
		DeleteUserData(user)
		return 0, fmt.Errorf("Failed to hash password cleaning user and sending 500: %v", err)
	}

	pass := UserPassword{
		Uid:        user.Uid,
		HashedPass: string(hashedPass),
	}

	_, err = AddUserPass(pass)
	if err != nil {
		return 0, fmt.Errorf("unable to add test user: %v", err)
	}

	return int(uid), nil
}

func deleteTestUser() error {
	// Clean database
	user, err := GetUserData(testUser.Email)
	if err != nil {
		return fmt.Errorf("failed to fetch created image data: %v", err)
	}
	err = DeleteUserData(user)
	if err != nil {
		return fmt.Errorf("failed to delete created user data: %v", err)
	}
	return nil
}

/*
func testBody (t *testing.T) {
	func TestAuth(t *testing.T) {

	// Create testUser
	_, err := createTestUser()
	if err != nil {
		t.Errorf("failed to create test user: %v", err)
	}

	// Configure http message
	router := configureRoutes()

	// Request recorder init
	rr := httptest.NewRecorder()

	// Configure http request
	req, err := http.NewRequest("GET", "/auth", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup database
	err = deleteTestUser()
	if err != nil {
		t.Errorf("failed to delete test user: %v", err)
	}
}
}
*/
