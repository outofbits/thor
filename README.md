# THOR MONITORING

Thor is a tool for monitoring a swarm of jormungandr nodes and keeping them all together in sync with the blockchain. A node is continuously compared to the latest tip reported by other nodes and if a node falls behind a specified number `x` of blocks or the creation date of the most recent received block lies more than x minutes in the past, then a specified action is taken (e.g. shut down, email report, etc.).

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

You should have a setup of your jormungandr node in which it automitically restarts after a shut down (e.g. systemd, docker compose/swarm, kubernetes). This is expected by this tool.

## Leader Jury
The leader jury gets the node statistics for every node in each interval and the jury remembers the last `window` node stats. The memory is then used to continiously compute the health of all nodes. The health of a node is specified at the moment by how little it drifted away from the maximum reported block height in this window. An exclusion zone can be specified such that no leader change can happen `exclusion_zone` seconds in front of a scheduled block. This mechanism shall prevent missed blocks as well as adversorial forks. 

The epoch turn over is a challenging task with the current design of Jormungandr. The current strategy is to stick for `turnover_exclusion_zone` seconds to the current leader and to shut down all the non leader nodes one after another (not all at once). This strategy can lead to missed block, if the leader fails. However, a missed block is better than creating an  adversorial fork and being publicly shamed.

| Name | Description | Default |
|---|---| ---- |
| cert | path to the  node-secret YAML configuration file | -no default- |
| window | number of checkpoints (occur in the frequency of `interval`ms) that shall be considered for the health metric | 5 |
| exclusion_zone | number of seconds in front of a sheduled block in which no leader change is allowed | 10 |
| turnover_exclusion_zone | number of seconds after a turn over in which no leader change is allowed | 600 |

```
monitor:
  interval: 1000
  leader_jury:
    cert: node-secret.yaml
    window: 8
    exclusion_zone: 10
```

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

## Feedback

If you have any suggestions, feel free to open an issue or submit a pull request.

TICKER: SOBIT
