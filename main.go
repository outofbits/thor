package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/sobitada/thor/config"
	"github.com/sobitada/thor/monitor"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const ApplicationName string = "thor"

func printUsage() {
	fmt.Printf("Usage:\n\t%v <config>\n\nArguments:\n\tconfig - YAML configuration for this thor instance.\n\n", ApplicationName)
}

func setLoggingConfiguration(config config.Config) {
	level, err := log.ParseLevel(config.Logging.Level)
	if err == nil {
		log.SetLevel(level)
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	args := os.Args[1:]
	if len(args) == 1 {
		data, err := ioutil.ReadFile(args[0])
		if err == nil {
			var conf config.Config
			err = yaml.UnmarshalStrict(data, &conf)
			if err == nil {
				setLoggingConfiguration(conf)
				nodes := config.GetNodesFromConfig(conf)
				if len(nodes) > 0 {
					m := monitor.GetNodeMonitor(nodes, config.GetNodeMonitorBehaviour(conf))
					m.RegisterAction(monitor.ShutDownAction{})
					m.Watch()
				} else {
					fmt.Printf("No passive/leader nodes specified. Nothing to do.")
					os.Exit(0)
				}
			} else {
				fmt.Printf("Could not parse the config file. %s", err.Error())
				os.Exit(1)
			}
		} else {
			fmt.Printf("Could not parse the config file. %s", err.Error())
			os.Exit(1)
		}
	} else {
		printUsage()
		os.Exit(1)
	}
}
