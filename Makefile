ENV=dev
APIURL=http://127.0.0.1
MQ=127.0.0.1
DOCKERFILE=Dockerfile
BRIDGE=brtest
GATEWAY_IP=172.19.0.1
SUBNET=$(GATEWAY_IP)/24
BA_PREFIX=bytearena/
cmd=cmd/arena-trainer
go=/usr/bin/go
build_args=

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

clean:
	rm -fv cmd/arena-trainer/arena-trainer

build-go: clean
	$(go) version
	cd $(cmd) && $(go) build $(build_args) -ldflags="-s -w"
	strip --strip-all --strip-dwo --strip-unneeded --remove-section=.note.gnu.gold-version --remove-section=.comment --remove-section=.note --remove-section=.note.gnu.build-id --remove-section=.note.ABI-tag cmd/arena-trainer/arena-trainer
	du -sh cmd/arena-trainer/arena-trainer

build-gccgo: clean
	$(go) version
	cd $(cmd) && $(go) build $(build_args) \
		-compiler gccgo \
		-ldflags="-s -w" \
		-gccgoflags "-O3 -s -W -gno-column-info -g0 -gno-pubnames -gno-record-gcc-switches -gno-split-dwarf -gstrict-dwarf -fcode-hoisting -fopt-info -fomit-frame-pointer -fno-exceptions -fno-asynchronous-unwind-tables -fno-unwind-tables"
	strip --strip-all --strip-dwo --strip-unneeded --remove-section=.note.gnu.gold-version --remove-section=.comment --remove-section=.note --remove-section=.note.gnu.build-id --remove-section=.note.ABI-tag cmd/arena-trainer/arena-trainer
	du -sh cmd/arena-trainer/arena-trainer

sstrip:
	~/Documents/BR903/ELFkickers/bin/sstrip cmd/arena-trainer/arena-trainer

dump:
	strings cmd/arena-trainer/arena-trainer > out

install:
	sudo mv -v cmd/ba/ba /usr/local/bin/ba
