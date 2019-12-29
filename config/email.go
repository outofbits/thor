package config

import (
    "fmt"
    "github.com/sobitada/thor/monitor"
    "net/smtp"
)

// email config for this application
type Email struct {
    // email from which messages shall be sent.
    SourceEmail string `yaml:"source"`
    // a list of Emails to which all messages shall be sent.
    DestinationEmails []string `yaml:"destinations"`
    // server for the SMTP server that shall be used for
    // sending the messages.
    SMTPServer struct {
        // host address of the SMTP server.
        Host string `yaml:"host"`
        // port number of the SMTP server.
        Port uint16 `yaml:"port"`
        // authentication config for the SMTP server.
        Authentication struct {
            Username string `yaml:"username"`
            Password string `yaml:"password"`
        } `yaml:"authentication"`
    } `yaml:"server"`
}

// parses the email configuration and returns the email action config that
// is required by email actions. if this config cannot be parsed, an error
// will be returned.
func ParseEmailConfiguration(conf General) (*monitor.EmailActionConfig, error) {
    if conf.Email != nil {
        email := *conf.Email
        if email.SMTPServer.Host != "" && email.SMTPServer.Port != 0 {
            if email.SMTPServer.Authentication.Username != "" && email.SMTPServer.Authentication.Password != "" {
                emailConf := monitor.EmailActionConfig{
                    SourceAddress:        email.SourceEmail,
                    DestinationAddresses: email.DestinationEmails,
                    ServerURL:            fmt.Sprintf("%v:%v", email.SMTPServer.Host, email.SMTPServer.Port),
                    Authentication: smtp.PlainAuth("", email.SMTPServer.Authentication.Username,
                        email.SMTPServer.Authentication.Password, email.SMTPServer.Host),
                }
                return &emailConf, nil
            } else {
                return nil, ConfigurationError{Path: "email/server/authenticaiton", Reason: "Username and password for the SMTP server must be specified."}
            }
        } else {
            return nil, ConfigurationError{Path: "email/server", Reason: "Host and port must be specified for the server."}
        }
    }
    return nil, nil
}
