#!/bin/sh
docker run -it --rm -e POSTGRES_PASSWORD=api-password -e POSTGRES_USER=api -e POSTGRES_DB=api -p 5432:5432 postgres:latest