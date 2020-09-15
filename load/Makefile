BASEDIR = $(shell pwd)
REDISNAME = load-redis

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