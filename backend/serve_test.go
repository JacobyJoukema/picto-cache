package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type RouteTest struct {
	Route    string
	Func     func(w http.ResponseWriter, req *http.Request)
	Method   []string
	Expected []int
}

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

	user := User{
		Firstname: "Jacoby",
		Lastname:  "Joukema",
		Email:     "user@mail.com",
	}
	userPass := "pass"

	// Generate incomplete multipart form data
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)

	err = writer.WriteField("firstname", user.Firstname)
	if err != nil {
		t.Errorf("failed to create form field: %v", err)
	}
	err = writer.WriteField("lastname", user.Lastname)
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
	err = writer.WriteField("email", user.Email)
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

	// Clean database
	user, err = GetUserData(user.Email)
	if err != nil {
		t.Errorf("failed to fetch created image data: %v", err)
	}
	err = DeleteUserData(user)
	if err != nil {
		t.Errorf("failed to delete created user data: %v", err)
	}

}

// testAuth tests the /auth endpoint for a valid and an invalid credential
func testAuth(t *testing.T) {
	/*
		// Configure http message
		router := configureRoutes()

		// Request recorder init
		rr := httptest.NewRecorder()

		// Configure http request
		req, err := http.NewRequest("GET", "/ping", nil)
		if err != nil {
			t.Fatal(err)
		}

		// populate db with uid: 0 password: test
		// Attempt to hash password for storage
		hashedPass, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
		if err != nil {
			t.Errorf("failed to hash password cleaning user and sending 500: %v", err)
		}

		pass := UserPassword{
			Uid:        0,
			HashedPass: string(hashedPass),
		}

		// Add hashed password to password table
		_, err = AddUserPass(pass)
		if err != nil {
			t.Errorf("failed to store hashed password cleaning user and sending 500: %v", err)
		}*/
}

/*
func testBody (t *testing.T) {
	// Configure http message
	router := configureRoutes()

	// Request recorder init
	rr := httptest.NewRecorder()

	// Configure http request
	req, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
}
*/
