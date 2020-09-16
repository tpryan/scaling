BASEDIR = $(shell pwd)
REDISNAME = load-redis
APPNAME = loadreceiver

.DEFAULT_GOAL := app

app: clean loadreceiver build serve

loadreceiver:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o loadreceiver main.go

build:
	docker build -t $(APPNAME) "$(BASEDIR)/."

serve:
	docker run --name=$(APPNAME) -e REDISPORT=6379 -e REDISHOST=docker.for.mac.localhost -e SCALE_ENV=Local -d -P -p 8081:8080 $(APPNAME)

clean:
	-rm $(APPNAME)
	-docker stop $(APPNAME)
	-docker rm $(APPNAME)
	-docker rmi $(APPNAME)

redis: redisclean
	docker run --name $(REDISNAME) -p 6379:6379 -d redis	

redisclean:
	-docker stop $(REDISNAME)
	-docker rm $(REDISNAME)

dev: 
	(trap 'kill 0' SIGINT; \
	export REDISHOST=127.0.0.1 && \
	export REDISPORT=6379 && \
	go run main.go )			