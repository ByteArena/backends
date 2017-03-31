# Byte Arena

![](https://cloud.githubusercontent.com/assets/4974818/24494371/57a8073c-1532-11e7-9026-469640cea9a7.png)
## Install

```bash
$ cd ~/go/src
$ mkdir -p github.com/netgusto
$ cd github.com/netgusto
$ git clone https://github.com/netgusto/bytearena
$ cd bytearena/agents/seeker
$ npm install
```

## Run

Requires docker and golang.

Replace `$LOCALIP` with your local LAN IP.

```bash
$ cd ~/go/src/github.com/netgusto/cmd/arena

$ # First time only
$ go get ./...

$ # Build and run
$ go build && HOST=$LOCALIP PORT=8888 TPS=10 AGENTS=2 AGENTIMP=seeker ./arena
# ctrl-c to tear down
```

Options:
* `TPS`: Turns per second
* `AGENTS`: Number of agents to spawn (one container each)
* `AGENTIMP`: Implementation of Agent (for the moment, subdir of /agents)
