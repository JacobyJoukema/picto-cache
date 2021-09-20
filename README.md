# Picto Cache
Picto Cache is a digital photo album and sharing platform built for the Shopify 2022 Winter Internship Application by Jacoby Joukema

## Overview
This system is designed with a heavy focus on the backend api and micro-service development principles. For this reason the frontend is simplistic in nature and is built to demonstrate the backend capabilities not for user experience.

There are three ways to interact with this system:

1. Use the online live demo that can be found here (TODO Deploy and link)
2. Interact with the live demo API via Swagger (TODO Deploy and link) or through apps like postman at (TOD Deploy and link API)
3. Clone and run on your personal device. Not recommended due to large dependencies for PostgreSQL and Yarn for the front end

## Design

## Project Dependencies
This project leverages a few dependencies in order to ensure that the software is robust and well documented

### API Dependencies
1. [https://golang.org/doc/install](Go) (Golang) - REQUIRED - Follow instructions on the official site to install for your system
2. [https://www.postgresql.org/download/](PostgreSQL) - REQUIRED - Follow instructions on the official site or [/devops/psql/psql-install.sh](/devops/psql/psql-install.sh) to install on Ubuntu.
3. [https://swagger.io/docs/open-source-tools/swagger-ui/usage/installation/](Swagger) - Recommended - Follow instructions on official site, used for api documentation and manual testing of endpoints.
4. [https://www.postman.com/](Postman) - Recommended - Follow instructions on official site, used for manual testing of endpoints.

## References
The following references were utilized in order to develop key components of this program

- https://medium.com/@jacoby.joukema/managing-your-sql-database-in-go-with-structql-74f4ebafc062 - Guide for using StructQL (written by me)
- https://goswagger.io/ - Docs for goswagger
- https://freshman.tech/file-upload-golang/ - Parsing multipart forms in Go with file uploads
- https://golangcode.com/get-the-content-type-of-file/ - Identify file type in Go
- https://tutorialedge.net/golang/authenticating-golang-rest-api-with-jwts/ - JWT authentication in Golang
- https://dev.to/techschoolguru/how-to-securely-store-passwords-3cg7 - Password hashing and storage techniques for go

