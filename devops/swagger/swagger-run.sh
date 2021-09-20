#!/bin/bash

sudo docker run -p 8080:8080 -e SWAGGER_JSON=/tmp/api-spec.yaml -v `pwd`:/tmp swaggerapi/swagger-ui