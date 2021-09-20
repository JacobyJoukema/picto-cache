package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
)

type RouteTest struct {
	Route    string
	Func     func(w http.ResponseWriter, req *http.Request)
	Method   []string
	Expected []int
}

func setup() *mux.Router {
	// establish router
	router := mux.NewRouter()

	// add routes
	// Basic service endpoints
	router.HandleFunc("/", home).Methods("GET", "OPTIONS", "POST", "PUT", "DELETE")
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
	router.HandleFunc("/image/meta", imageMetaRequest).Methods("GET", "OPTIONS")

	return router
}

// TestRouting evaluates a number of endpoints without authentication and ensures the correct response headers
// This is a catch all for routing detailed tests of endpoint edge cases are completed in
// the appropriate test function.
func TestRouting(t *testing.T) {
	router := setup()

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
		setup()
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

	// Request recorder init
	rr := httptest.NewRecorder()

	// Define /ping request
	req, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Submit request
	handler := http.HandlerFunc(ping)
	handler.ServeHTTP(rr, req)

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
