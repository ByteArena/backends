# Byte Arena

![](https://cloud.githubusercontent.com/assets/4974818/24494371/57a8073c-1532-11e7-9026-469640cea9a7.png)
## Install

```bash
$ cd ~/go/src
$ mkdir -p github.com/netgusto
$ cd github.com/netgusto
$ git clone https://github.com/netgusto/bytearena

# install agents
$ cd bytearena/agents/seeker
$ npm install

# install go pkgs
$ cd ~/go/src/github.com/netgusto/bytearena
$ go get ./...
```

## Run

**Arena**

Requires docker and golang.

Replace `$LOCALIP` with your local LAN IP.

```bash
$ cd ~/go/src/github.com/netgusto/bytearena/cmd/arena
$ # Build and run
$ go build && HOST=$LOCALIP PORT=8888 TPS=10 AGENTS=2 AGENTIMP=seeker ./arena
# ctrl-c to tear down
```

Options:
* `TPS`: Turns per second
* `AGENTS`: Number of agents to spawn (one container each)
* `AGENTIMP`: Implementation of Agent (for the moment, subdir of /agents)

**Visualisation**

```bash
$ cd ~/go/src/github.com/netgusto/bytearena/cmd/streamderiver
$ # Build and run
$ go build && ./streamderiver
# ctrl-c to tear down
# http://yourip:8080 to display the web visualisation (WIP; click open)
```
