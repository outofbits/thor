package main

import (
    "flag"
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/thor/config"
    "github.com/sobitada/thor/monitor"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "os"
)

const ApplicationName string = "thor"
const ApplicationVersion string = "0.1.0-SNAPSHOT"

func printUsage() {
    fmt.Printf(`Usage:
  %v <config> or %v [-help | -version]

Arguments:
  <config>
        YAML configuration for this thor instance.
`, ApplicationName, ApplicationName)
    flag.PrintDefaults()
}

func printVersion() {
    fmt.Printf("%v (%v) 2020@SOBIT\n", ApplicationName, ApplicationVersion)
}

func setLoggingConfiguration(config config.General) {
    level, err := log.ParseLevel(config.Logging.Level)
    if err == nil {
        log.SetLevel(level)
    }
    log.SetFormatter(&log.TextFormatter{
        FullTimestamp: true,
    })
}

func main() {
    help := flag.Bool("help", false, "prints this usage message.")
    version := flag.Bool("version", false, "prints the version of this application.")
    flag.Parse()
    if *help {
        printUsage()
    } else if *version {
        printVersion()
    } else {
        args := flag.Args()
        if len(args) == 1 {
            data, err := ioutil.ReadFile(args[0])
            if err == nil {
                var conf config.General
                err = yaml.UnmarshalStrict(data, &conf)
                if err == nil {
                    setLoggingConfiguration(conf)
                    nodes := config.GetNodesFromConfig(conf)
                    if len(nodes) > 0 {
                        m := monitor.GetNodeMonitor(nodes, config.GetNodeMonitorBehaviour(conf))
                        m.RegisterAction(monitor.ShutDownWithBlockLagAction{})
                        if conf.PoolTool != nil {
                            poolToolConf := *conf.PoolTool
                            if poolToolConf.UserID != "" && poolToolConf.GenesisHash != "" && poolToolConf.PoolID != "" {
                                m.RegisterAction(monitor.PostLastTipToPoolToolAction{
                                    PoolID:      poolToolConf.PoolID,
                                    UserID:      poolToolConf.UserID,
                                    GenesisHash: poolToolConf.GenesisHash,
                                })
                            } else {
                                fmt.Print("Personal pool ID, pool tool user ID as well as genesis hash of the blockchain must be specified.")
                                os.Exit(1)
                            }
                        }
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
}
