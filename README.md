# Byte Arena

![](https://cloud.githubusercontent.com/assets/4974818/24494371/57a8073c-1532-11e7-9026-469640cea9a7.png)
## Install

```bash
$ cd "$GOPATH"/src
$ mkdir -p github.com/netgusto
$ cd github.com/netgusto
$ git clone git@github.com:netgusto/bytearena.git

# install go pkgs
$ cd "$GOPATH"/src/github.com/netgusto/bytearena
$ go get ./...

# install agents
$ cd agents/seeker
$ npm install

```

## Run

**Arena**

Requires docker and golang.

Replace `$LOCALIP` with your local LAN IP.

```bash
$ cd "$GOPATH"/src/github.com/netgusto/bytearena/cmd/arena
$ # Pull node image
$ docker create node
$ # Build and run
$ go build && HOST=$LOCALIP PORT=8888 TPS=8 AGENTS=8 AGENTIMP=seeker ./arena
# ctrl-c to tear down
```

Options:
* `TPS`: Turns per second
* `AGENTS`: Number of agents to spawn (one container each)
* `AGENTIMP`: Implementation of Agent (for the moment, subdir of /agents)

**Visualisation**

```bash
$ cd "$GOPATH"/src/github.com/netgusto/bytearena/cmd/streamderiver
$ # Build and run
$ go build && ./streamderiver
# ctrl-c to tear down
# http://yourip:8080 to display the web visualisation (WIP; click open)
```
