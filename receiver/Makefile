BASEDIR = $(shell pwd)
APPNAME = receiver
PROJECT=$(LOADPROJECT)
REGION=us-central1
ZONE=us-central1-c
REDISNAME = load-redis
REDISIP=$(shell gcloud beta redis instances describe $(REDISNAME) --region $(REGION) --format='value(host)')
ENDPOINT=$(shell gcloud run services list --platform managed --format='value(URL)' )
GKEIP=$(shell gcloud compute addresses list --filter='name=gkeloadip' --format='value(ADDRESS)' )

.DEFAULT_GOAL := app

env:
	gcloud config set project $(PROJECT)


app: clean receiver build serve

receiver:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o receiver main.go

build:
	docker build -t $(APPNAME) "$(BASEDIR)/."

serve:
	docker run --name=$(APPNAME) -e REDISPORT=6379 -e SCALE_ENV=Local \
	-e REDISHOST=docker.for.mac.localhost \
	-e ENDPOINT=http://docker.for.mac.localhost:8081/record  \
	-d -P -p 8081:8080 $(APPNAME)
	@echo ----------------------------------------------------	
	@echo Load Running at 127.0.0.1:8081
	@echo ----------------------------------------------------	
		
clean:
	-rm $(APPNAME)
	-docker stop $(APPNAME)
	-docker rm $(APPNAME)
	-docker rmi $(APPNAME)


deploy: cloudrun gke appengine

cleanexec:
	-rm receiver

appengine: env
	gcloud app deploy -q

cloudbuild: env cleanexec receiver
	gcloud builds submit

cloudrun: env cloudbuild
	gcloud run deploy receiver -q --image gcr.io/$(PROJECT)/receiver:latest \
	--set-env-vars=REDISHOST="$(REDISIP)",REDISPORT="6379",SCALE_ENV="cloudrun",ENDPOINT="$(ENDPOINT)/record" \
	--platform=managed --region=$(REGION)  --vpc-connector=load-redisconnector \
	--allow-unauthenticated

gkecreds:
	gcloud container clusters get-credentials loadcluster

gke: env gkecreds cloudbuild
	-kubectl delete service receiver		
	-kubectl delete deployment receiver
	kubectl apply -f deployment.yaml
	kubectl expose deployment receiver --type="NodePort" --port=8080 --target-port=8080 

gkenew: env gkecreds gkeclean cloudbuild
	kubectl apply -f deployment.yaml
	kubectl autoscale deployment receiver --cpu-percent=50 --max=150
	kubectl expose deployment receiver --type="NodePort" --port=8080 --target-port=8080 
	kubectl apply -f ingress.yaml	

cluster: env
	gcloud beta container clusters create "loadcluster" --zone $(ZONE) \
	--no-enable-basic-auth --cluster-version "1.17.9-gke.6300" \
	--machine-type "e2-medium" --image-type "COS" \
	--disk-type "pd-standard" --disk-size "100" \
	--tags "default-allow-http" \
	--metadata disable-legacy-endpoints=true \
	--num-nodes "3" --enable-stackdriver-kubernetes --enable-ip-alias \
	--default-max-pods-per-node "110" --no-enable-master-authorized-networks \
	--addons HorizontalPodAutoscaling,HttpLoadBalancing --enable-autoupgrade \
	--enable-autorepair --max-surge-upgrade 1 --max-unavailable-upgrade 0

gkeclean:
	-kubectl delete ingress basic-ingress
	-kubectl delete service receiver
	-kubectl delete hpa receiver
	-kubectl delete deployment receiver

deletecluster:
	gcloud beta container clusters delete "loadcluster" -q

ip:
	gcloud compute addresses create gkeloadip --global

deploy: env clean appengine cloudrun 

printip:
	@echo CLOUDRUN $(ENDPOINT)/register 
	@echo REDISIP  $(REDISIP)/register 
	@echo GKEIP    $(GKEIP)/register 

schedule:
	gcloud scheduler jobs create http register-gke --schedule="* * * * *" \
	--uri=http://$(GKEIP)/register 
	gcloud scheduler jobs create http register-run --schedule="* * * * *" \
	--uri=$(ENDPOINT)/register 
	gcloud scheduler jobs create app-engine register-gae --schedule="* * * * *" \
	--service receiver --relative-url /register
