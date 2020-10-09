BASEDIR = $(shell pwd)
REDISNAME = load-redis
REGION=us-central1
PROJECT=$(LOADPROJECT)

.DEFAULT_GOAL := dev

env:
	gcloud config set project $(PROJECT)


redis: redisclean
	docker run --name $(REDISNAME) -p 6379:6379 -d redis
	@echo ----------------------------------------------------	
	@echo Redis Running at 127.0.0.1:6379	
	@echo ----------------------------------------------------		

redisclean:
	-docker stop $(REDISNAME)
	-docker rm $(REDISNAME)


clean: redisclean
	cd loadgen && $(MAKE) clean
	cd load && $(MAKE) clean
	cd visualizer && $(MAKE) clean

dev: 
	@cd loadgen && $(MAKE) 
	@cd load && $(MAKE) 
	@cd visualizer && $(MAKE) 

scratch: redis dev	





services: env
	-gcloud services enable vpcaccess.googleapis.com
	-gcloud services enable cloudbuild.googleapis.com
	-gcloud services enable run.googleapis.com
	-gcloud services enable appengine.googleapis.com 
	-gcloud services enable compute.googleapis.com 
	-gcloud services enable redis.googleapis.com
	-gcloud services enable cloudscheduler.googleapis.com



memorystore: env
	-gcloud redis instances create $(REDISNAME) --size=1 --region=$(REGION)
	-gcloud compute networks vpc-access connectors create \
		$(REDISNAME)connector --network default --region $(REGION) \
		--range 10.8.0.0/28 