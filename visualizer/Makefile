BASEDIR = $(shell pwd)
REDISNAME = load-redis
APPNAME = visualizer
PROJECT=$(LOADPROJECT)

.DEFAULT_GOAL := app

env:
	gcloud config set project $(PROJECT)

app: clean visualizer build serve

visualizer:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o visualizer main.go

build:
	docker build -t $(APPNAME) "$(BASEDIR)/."

serve:
	docker run --name=$(APPNAME) -e REDISPORT=6379 -e REDISHOST=docker.for.mac.localhost -e SCALE_ENV=Local -d -P -p 8080:8080 $(APPNAME)
	@echo ----------------------------------------------------	
	@echo Visualizer Running at 127.0.0.1:8080	
	@echo ----------------------------------------------------	

clean:
	-rm $(APPNAME)
	-docker stop $(APPNAME)
	-docker rm $(APPNAME)
	-docker rmi $(APPNAME)

