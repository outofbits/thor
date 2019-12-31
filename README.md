# THOR MONITORING

Thor is a tool for monitoring a swarm of jormungandr nodes and keeping them all together in sync with the blockchain. A 
node is continuously compared to the latest tip reported by other nodes and if a node
falls behind a specified number `x` of blocks, then a specified action is taken (e.g. shut down, logging, email report, etc.).

## Build
You need to have the Go language installed on your machine; instructions are [here](https://golang.org/doc/install#install). Then
you have to fetch the source code of this repository, which can be done with the following command.

```
go get github.com/sobitada/thor
```

Then you must switch to your GO home (often located in your home directory per default) and go to the
fetched source code. There you can build this program with the following two steps. First all used libraries
are fetched and then the build process is started.
```
go get .
```
```
go build
```

Afterwards you should see an executable named `thor` for your OS and architecture.

## Run
```
Usage:
	thor <config>

Arguments:
	config - YAML configuration for this thor instance.
```

The configuration looks like this. You can specify the swarm of peers and the interval in which their status shall be
checked as well as the logging level.
```
logging:
  level: info
blockchain:
  genesisBlockHash: "8e4d2a343f3dcf9330ad9035b3e8d168e6728904262f2c434a4f8f934ec7b676"
  genesisBlockTime: "2019-12-13T19:13:37+00:00"
  slotsPerEpoch: 43200
  slotDuration: 2000
peers:
  - name: "Local 1"
    api: http://jormungandr-1:3101
    maxBlockLag: 10
  - name: "Local 2"
    api: http://jormungandr-2:3101
    maxBlockLag: 10
    maxTimeSinceLastBlock: 60000
  - name: "Local 3"
    api: http://jormungandr-3:3101
    maxTimeSinceLastBlock: 60000
monitor:
  interval: 1000
```

At the moment only one action is available and that is "shut down". You should have a setup of your jormungandr node in
which it automitically restarts after a shut down (e.g. systemd, docker compose/swarm, kubernetes).

## Feedback

If you have any suggestions, feel free to open an issue or submit a pull request.

TICKER: SOBIT
