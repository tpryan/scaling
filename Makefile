BASEDIR = $(shell pwd)
REDISNAME = load-redis

.DEFAULT_GOAL := dev

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