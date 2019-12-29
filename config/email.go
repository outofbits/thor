package config

import (
    "github.com/sobitada/thor/monitor"
    "net/smtp"
    "net/url"
)

type Email struct {
    SourceEmail       string   `yaml:"source"`
    DestinationEmails []string `yaml:"destinations"`
    Authentication    struct {
        Username string `yaml:"username"`
        Password string `yaml:"password"`
    } `yaml:"authentication"`
    Server struct {
        URL url.URL `yaml:"url"`
    } `yaml:"server"`
}

func ParseEmailActions(conf General) (*monitor.EmailActionConfig, error) {
    if conf.Email != nil {
        email := *conf.Email
        if email.Authentication.Username != "" && email.Authentication.Password != "" {
            emailConf := monitor.EmailActionConfig{
                SourceAddress:        email.SourceEmail,
                DestinationAddresses: email.DestinationEmails,
                Authentication:       smtp.PlainAuth("", email.Authentication.Username, email.Authentication.Password, email.Server.URL.String()),
            }
            return &emailConf, nil
        } else {
            return nil, ConfigurationError{Path: "email/authenticaiton", Reason: "Username and password for the SMTP server must be specified."}
        }
    }
    return  nil, nil
}
