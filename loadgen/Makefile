# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
BASEDIR = $(shell pwd)

.DEFAULT_GOAL := app

app: clean abrunner build serve

abrunner:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o abrunner main.go

build:
	docker build -t abrunner "$(BASEDIR)/."

serve:
	docker run --name=abrunner -e REDISPORT=6379 -e REDISHOST=docker.for.mac.localhost -d -P -p 8082:8080 abrunner
	@echo ----------------------------------------------------	
	@echo LoadGen Running at 127.0.0.1:8082	
	@echo ----------------------------------------------------	
	

clean:
	-rm abrunner
	-docker stop abrunner
	-docker rm abrunner
	-docker rmi abrunner