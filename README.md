# THOR MONITORING

Thor is a tool for monitoring a swarm of jormungandr nodes and keeping them all together in sync with the blockchain. A 
node is continuously compared to the latest tip reported by other nodes and if a node
falls behind a specified number `x` of blocks, then a specified action is taken (e.g. shut down, logging, email report, etc.).

![The last battle of Thor](docs/images/thor-jormungandr.jpg)

Credit for Image: [Sasin](https://www.deviantart.com/sasin)

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
peers:
  - name: "eu-central-1"
    api: http://10.0.0.2:3031
    maxBlockLag: 10
  - name: "us-south-east-1"
    api: http://10.0.0.3:3031
    maxBlockLag: 10
monitor:
  interval: 1000 # in ms
```

At the moment only one action is available and that is "shut down". You should have a setup of your jormungandr node in
which it automitically restarts after a shut down (e.g. systemd, docker compose/swarm, kubernetes).

## Feedback

If you have any suggestions, feel free to open an issue or submit a pull request.

TICKER: SOBIT
