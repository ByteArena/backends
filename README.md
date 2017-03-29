# bytearena
Byte Arena

## Install

```bash
$ cd ~/go/src
$ mkdir -p github.com/netgusto
$ cd github.com/netgusto
$ git clone https://github.com/netgusto/bytearena
$ cd bytearena/client
$ npm install
```

## Run

Requires docker and golang.

Replace `$LOCALIP` with your local LAN IP.

```bash
$ cd ~/go/src/github.com/netgusto/server
$ # First time only
$ go get ./...
$ # Build and run
$ go build && HOST=$LOCALIP PORT=8888 TPS=10 AGENTS=2 ./server
// ctrl-c to tear down
```

Options:
* `TPS`: Turns per second
* `AGENTS`: Number of agents to spawn (one container each)
