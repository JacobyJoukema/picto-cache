#!/bin/bash

# Testing credentials only production must use production credentials and db
sudo -u postgres psql -c "CREATE USER tester PASSWORD 'testpass';"
sudo -u postgres psql -c "CREATE DATABASE dbtest;"
sudo service postgresql start