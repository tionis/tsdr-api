#!/bin/sh
docker run -it --rm -e POSTGRES_PASSWORD=api-password -e POSTGRES_USER=api -e POSTGRES_DB=api -p 5432:5432 postgres:latest
#docker run --name dev-postgres -e POSTGRES_PASSWORD=password -e POSTGRES_USER=test-user -e POSTGRES_DB -p 5432:5432 -d postgres