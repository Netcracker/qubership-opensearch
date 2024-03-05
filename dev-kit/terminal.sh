#!/usr/bin/env bash

docker-compose up --build -d
winpty docker-compose exec operator-sdk bash
docker-compose down
