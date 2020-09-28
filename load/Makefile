BASEDIR = $(shell pwd)
APPNAME = loadreceiver
PROJECT=$(LOADPROJECT)
REGION=us-central1
REDISNAME = load-redis
REDISIP=$(shell gcloud beta redis instances describe $(REDISNAME) \
			--region $(REGION) --format='value(host)')
ENDPOINT=$(shell gcloud run services list --platform managed --format='value(URL)')			

.DEFAULT_GOAL := app

env:
	gcloud config set project $(PROJECT)


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


appengine:
	gcloud app deploy -q

cloudbuild: env
	gcloud builds submit

cloudrun: 
	gcloud run deploy receiver -q --image gcr.io/$(PROJECT)/load:latest \
	--set-env-vars=REDISHOST="$(REDISIP)",REDISPORT="6379",SCALE_ENV="cloudrun",ENDPOINT="$(ENDPOINT)" \
	--platform=managed --region=$(REGION)  --vpc-connector=load-redisconnector \
	--allow-unauthenticated
