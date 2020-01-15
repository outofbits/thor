package main

import (
    "flag"
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/thor/config"
    "github.com/sobitada/thor/monitor"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "os"
)

const ApplicationName string = "thor"
const ApplicationVersion string = "0.2.0-alpha1"

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
                        var blockChainSettings *cardano.TimeSettings = nil
                        if conf.Blockchain != nil {
                            blockChainSettings, err = config.GetTimeSettings(*conf.Blockchain)
                            if err != nil {
                                log.Warnf("Could not parse the time settings of blockchain. %v", err.Error())
                            }
                        }
                        // Pool Tool
                        poolTool, err := config.ParsePoolToolConfig(conf)
                        if err != nil {
                            fmt.Print(err.Error())
                            os.Exit(1)
                        }
                        if poolTool != nil {
                            go poolTool.Start()
                        }
                        // Leader Jury
                        leaderJury, err := config.GetLeaderJury(nodes, blockChainSettings, conf)
                        if err == nil {
                            if leaderJury != nil {
                                go leaderJury.Judge()
                            }
                        } else {
                            log.Warnf("Leader jury was not configured correctly. %v", err.Error())
                        }
                        // Monitor
                        m := monitor.GetNodeMonitor(nodes, config.GetNodeMonitorBehaviour(conf), parseActions(conf),
                            blockChainSettings, poolTool, leaderJury)
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

func parseActions(conf config.General) []monitor.Action {
    actions := make([]monitor.Action, 0)
    actions = append(actions, monitor.ShutDownWithBlockLagAction{})
    actions = append(actions, monitor.ShutDownWhenStuck{})
    // parse pool tool action configuration

    // parse email action configuration.
    emailActionConfig, err := config.ParseEmailConfiguration(conf)
    if err == nil {
        if emailActionConfig != nil {
            actions = append(actions, monitor.ReportBlockLagPerEmailAction{Config: *emailActionConfig})
            actions = append(actions, monitor.ReportStuckPerEmailAction{Config: *emailActionConfig})
        }
    } else {
        fmt.Print(err.Error())
        os.Exit(1)
    }
    return actions
}
