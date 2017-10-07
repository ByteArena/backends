APIURL=http://127.0.0.1
MQ=127.0.0.1
DOCKERFILE=Dockerfile

build-arenamaster:
	docker build -f docker/arena-master/$(DOCKERFILE) -t arenamaster .

build-arenaserver:
	docker build -f docker/arena-server/$(DOCKERFILE) -t arenaserver .

build-linuxkit:
	make -C ~/go/src/github.com/bytearena/linuxkit build

run-arenamaster:
	docker run -it --privileged -e APIURL=$(APIURL) -e MQ=$(MQ) --net host -v ~/go/src/github.com/bytearena/linuxkit/linuxkit.raw:/linuxkit.raw arenamaster
