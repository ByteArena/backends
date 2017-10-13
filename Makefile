APIURL=http://127.0.0.1
MQ=127.0.0.1
DOCKERFILE=Dockerfile
BRIDGE=brtest
GATEWAY_IP=172.19.0.1
SUBNET=$(GATEWAY_IP)/24

build:
	cd cmd && bash buildall.sh

build-arenamaster:
	docker build \
		-f docker/arena-master/$(DOCKERFILE) \
		--build-arg BRIDGE=$(BRIDGE) \
		-t arenamaster .

build-arenaserver:
	docker build \
		-f docker/arena-server/$(DOCKERFILE) \
		-t arenaserver .

build-linuxkit:
	make -C ~/go/src/github.com/bytearena/linuxkit build

run-arenamaster:
	docker run -it --privileged -e APIURL=$(APIURL) -e MQ=$(MQ) --net host -v ~/go/src/github.com/bytearena/linuxkit/linuxkit.raw:/linuxkit.raw -v /lib/modules:/lib/modules arenamaster

create-br:
	brctl addbr $(BRIDGE)
	ifconfig $(BRIDGE) $(SUBNET) up

run-mq:
	docker run -p 6379:6379 redis

test:
	go test -v `go list ./... | grep -v /vendor/`
