ENV=dev
APIURL=http://127.0.0.1
MQ=127.0.0.1
DOCKERFILE=Dockerfile
BRIDGE=brtest
GATEWAY_IP=172.19.0.1
SUBNET=$(GATEWAY_IP)/24
BA_PREFIX=bytearena/

build:
	cd cmd && bash buildall.sh

build-arenamaster:
	docker build \
		-f docker/arena-master/$(DOCKERFILE) \
		--build-arg BRIDGE=$(BRIDGE) \
		-t $(BA_PREFIX)arenamaster .

build-arenaserver:
	docker build \
		-f docker/arena-server/$(DOCKERFILE) \
		-t $(BA_PREFIX)arenaserver .

build-linuxkit:
	make -C ~/Documents/Bytearena/ansible-deploy generate-arenaserver-image

run-arenamaster:
	docker run \
		-it \
		--privileged \
		-e APIURL=$(APIURL) \
		-e MQ=$(MQ) \
		-e ENV=$(ENV) \
		--net host \
		-v ~/Documents/Bytearena/ansible-deploy/linuxkit/linuxkit.raw:/linuxkit.raw \
		-v /lib/modules:/lib/modules \
		-v $(CURDIR)/data/log:/var/log/ \
		$(BA_PREFIX)arenamaster

create-br:
	brctl addbr $(BRIDGE)
	ifconfig $(BRIDGE) $(SUBNET) up

run-mq:
	docker run -p 6379:6379 redis

test:
	go test -v `go list ./... | grep -v /vendor/`
