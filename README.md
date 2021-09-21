# Picto Cache
Picto Cache is a digital photo album and sharing API built for the Shopify 2022 Winter Internship Application by Jacoby Joukema availble at [https://pictocache.jacobyjoukema.com](https://pictocache.jacobyjoukema.com)

## Overview
This system is designed in go with a heavy focus on the backend RESTfull API and micro-service principles. The focus of this project was to design a robust and well designed API complete with industry standard api authentication and validation techniques, for this reason the actual application lacks features but his highly extendable and very easy to interact with through tools detailed below.

There are three ways to interact with this system:

1. Use the online Swagger docs at [https://jacobyjoukema.com](https://jacobyjoukema.com) to interact with the live demo
2. Interact with the live demo API at [https://pictocache.jacobyjoukema.com](https://pictocache.jacobyjoukema.com) with an http tool of your preference
3. Use the shared Postman workspace to interact with the live demo or a local instance: [![Run in Postman](https://run.pstmn.io/button.svg)](https://app.getpostman.com/run-collection/9043989-40ff35a4-5f77-47d8-a108-992eff445614?action=collection%2Ffork&collection-url=entityId%3D9043989-40ff35a4-5f77-47d8-a108-992eff445614%26entityType%3Dcollection)
4. Clone and run on your personal device. Not recommended due development requirements such as Go and PostgeSQL

## Design
Picto Cache is a RESTfull API developed in Go. It is highly robust and features password hashing, token authentication, image data/meta storage, and user permissions. Picto Cache does not require any runtime online dependencies or services therefore it can be deployed on LAN for highly confidential information management.

### API
The api is documented in detail at [https://jacobyjoukema.com](https://jacobyjoukema.com). It was designed to be stateless and handle individual requests independently. This allows for a highly scalable API compatible with deployment management systems like Kubernetes if required.

### Data Model
All metadata is stored via PostgreSQL enabling highly efficient data retrieval and storage. Further, a SQL datastore allows for the scaling of response handlers without requiring the scaling of database resources. All interaction with the database is handled by [./backend/store.go](backend/store.go) using [https://pkg.go.dev/github.com/inflowml/structql](StructQl) - A Go packaged Co-designed by me that simplifies the management of SQL databases using struct tags in go.

#### Tables
The app instantiates and manages three SQL tables summarized by their [https://pkg.go.dev/github.com/inflowml/structql](StructQl) tags below

1. image_meta
```go
type Image struct {
	Id        int32  `json:"id" sql:"id" typ:"SERIAL" opt:"PRIMARY KEY"`
	Uid       int32  `json:"uid" sql:"uid"`
	Title     string `json:"title" sql:"title"`
	Ref       string `json:"ref" sql:"ref"`
	Size      int32  `json:"size" sql:"size"`
	Encoding  string `json:"encoding" sql:"encoding"`
	Shareable bool   `json:"shareable" sql:"shareable"`
}
```
2. user_meta
```go
type User struct {
	Uid       int32  `json:"uid" sql:"id" typ:"SERIAL" opt:"PRIMARY KEY"`
	Firstname string `json:"firstname" sql:"firstname"`
	Lastname  string `json:"lastname" sql:"lastname"`
	Email     string `json:"email" sql:"email"`
}
```
3. user_pass
```go
type UserPassword struct {
	Uid        int32  `sql:"id" opt:"PRIMARY KEY"` // Corresponds to User Uid
	HashedPass string `sql:"hashed_pass"`
}
```

### Testing

#### Methodology
In order to fully test this system a combination of unit and manual tests are required. It is impossible to get full unit testing coverage because networking systems are often unpredictable. For example the unit tests need a PostgreSQL test database running to properly evaluate the system effectiveness, therefore it is non-trivial to unit test the availability of the database without adding an additional testing layer on top of the system. Futher, the system was designed to sanitize incoming data however users may still attempt to circumvent these through a number of methods that can't be predicted and therefore full test coverage is very difficult

#### Unit Testing
All endpoints are tested for various valid and invalid calls through serve_test.go. This file also evaluates the effectiveness of store.go as those functions are used within serve.go and are internal facing. To run unit tests navigate to [./backend](/backend) and run go test.

#### Manual Testing
Manual testing is conducted through a number of tools including Swagger, Postman, and network browsers. See the API section for more details on manually testing and using the software.

## Installation

### Project Dependencies
This project leverages a few dependencies in order to ensure that the software is robust and well documented

1. [Go](https://golang.org/doc/install) (Golang) - REQUIRED - Follow instructions on the official site to install for your system
2. [PostGreSQL](https://www.postgresql.org/download/) - REQUIRED - Follow instructions on the official site or [/devops/psql/psql-install.sh](/devops/psql/psql-install.sh) to install on Ubuntu.
3. [Swagger](https://swagger.io/docs/open-source-tools/swagger-ui/usage/installation/) - Recommended - Follow instructions on official site, used for api documentation and manual testing of endpoints.
4. [Postman](https://www.postman.com/) - Recommended - Follow instructions on official site, used for manual testing of endpoints.

### Step by Step (Unix Comd Line)
1. Clone git repo `git clone https://github.com/JacobyJoukema/picto-cache.git`
2. Set up PostgreSQL testing database
```bash
    cd devops/psql
    ./psql-run
```
3. Run unit tests
```bash
    cd ../../backend
    go test .
```
4. Run go server
```bash
    go run .
```

### Environment Variables
The following environment variables are used to define system properties for deployments. When left unset server defaults to test parameters
- SIGNING_KEY - Server side key for encoding jwts
- REF_URL - Address of url used for image referencing ex. pictocache.jacobyjoukema.com
- GO_PORT - Port to serve http in the form of :PORT
- DB_NAME - Name of database
- DB_USER - Database username for this service
- DB_PASS - Database password for this user
- DB_HOST - Database host
- DB_PORT - Database port

## References
The following references were utilized in order to develop key components of this program

- https://medium.com/@jacoby.joukema/managing-your-sql-database-in-go-with-structql-74f4ebafc062 - Guide for using StructQL (written by me)
- https://goswagger.io/ - Docs for goswagger
- https://freshman.tech/file-upload-golang/ - Parsing multipart forms in Go with file uploads
- https://golangcode.com/get-the-content-type-of-file/ - Identify file type in Go
- https://tutorialedge.net/golang/authenticating-golang-rest-api-with-jwts/ - JWT authentication in Golang
- https://dev.to/techschoolguru/how-to-securely-store-passwords-3cg7 - Password hashing and storage techniques for go
- https://flaviocopes.com/golang-enable-cors/ - Supporting CORs in golang
- https://blog.questionable.services/article/testing-http-handlers-go/ - Testing http handlers in go
- https://golangbyexample.com/http-mutipart-form-body-golang/ - Multipart form client in golang

