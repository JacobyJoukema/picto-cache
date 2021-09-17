#!/bin/bash

echo "tester password: 'testpass'"
psql -h localhost -p 5432 -U tester -d dbtest