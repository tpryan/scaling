BASEDIR = $(shell pwd)
APPNAME = loadreceiver

.DEFAULT_GOAL := app

app: clean loadreceiver build serve

loadreceiver:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o loadreceiver main.go

build:
	docker build -t $(APPNAME) "$(BASEDIR)/."

serve:
	docker run --name=$(APPNAME) -e REDISPORT=6379 -e SCALE_ENV=Local \
	-e REDISHOST=docker.for.mac.localhost \
	-e ENDPOINT=docker.for.mac.localhost:8081  -d -P -p 8081:8080 $(APPNAME)
	@echo ----------------------------------------------------	
	@echo Load Running at 127.0.0.1:8081
	@echo ----------------------------------------------------	
		
clean:
	-rm $(APPNAME)
	-docker stop $(APPNAME)
	-docker rm $(APPNAME)
	-docker rmi $(APPNAME)
