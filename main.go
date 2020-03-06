package main

import (
    "flag"
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/thor/config"
    "github.com/sobitada/thor/leader"
    "github.com/sobitada/thor/monitor"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "os"
)

const ApplicationName string = "thor"
const ApplicationVersion string = "0.2.2-experimental"

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
            printProlog()
            data, err := ioutil.ReadFile(args[0])
            if err == nil {
                var conf config.General
                err = yaml.UnmarshalStrict(data, &conf)
                if err == nil {
                    setLoggingConfiguration(conf)
                    nodes, err := config.GetNodesFromConfig(conf)
                    if err == nil {
                        if len(nodes) > 0 {
                            timeSettings, err := config.GetTimeSettings(*conf.Blockchain)
                            if err != nil {
                                log.Warnf("Could not parse the time settings of blockchain. %v", err.Error())
                            }
                            // try to establish a schedule watchdog.
                            var watchdog *monitor.ScheduleWatchDog = nil
                            if timeSettings != nil {
                                watchdog = monitor.NewScheduleWatchDog(nodes, timeSettings)
                            } else {
                                log.Warnf("You have to set the time settings for the block chain for schedule watchdog.")
                            }
                            // try to establish the monitor.
                            nodeMonitor := monitor.GetNodeMonitor(nodes, config.GetNodeMonitorBehaviour(conf),
                                parseActions(), watchdog, timeSettings)
                            // try to establish the pool tool updater.
                            poolTool, err := config.ParsePoolToolConfig(nodeMonitor, conf)
                            if err != nil {
                                log.Warnf("The pool tool update could not be started. %v", err.Error())
                            }
                            // try to establish the prometheus client
                            prometheus, err := config.ParsePrometheusConfig(nodeMonitor, conf)
                            if err != nil {
                                log.Warnf("The Prometheus client could not be started. %v", err.Error())
                            }
                            // try to establish the leader jurry.
                            var leaderJurry *leader.Jury = nil
                            if timeSettings != nil {
                                leaderJurry, err = config.GetLeaderJury(nodes, nodeMonitor, watchdog, timeSettings, conf)
                                if err != nil {
                                    log.Errorf("Leader jury was not configured correctly. %v", err.Error())
                                }
                            } else {
                                log.Warnf("You have to set the time settings for the block chain for leader jury.")
                            }
                            // start all tools
                            if poolTool != nil {
                                go poolTool.Start()
                            }
                            if watchdog != nil {
                                go watchdog.Watch()
                            }
                            if leaderJurry != nil {
                                go leaderJurry.Judge()
                            }
                            if prometheus != nil {
                                go prometheus.Run()
                            }
                            nodeMonitor.Watch()
                        } else {
                            fmt.Printf("No passive/leader nodes specified. Nothing to do.")
                            os.Exit(0)
                        }
                    } else {
                        fmt.Printf("Peers cannot be parsed. %v", err.Error())
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

func parseActions() []monitor.Action {
    actions := make([]monitor.Action, 0)
    actions = append(actions, monitor.ShutDownWithBlockLagAction{})
    actions = append(actions, monitor.ShutDownWhenStuck{})
    return actions
}
