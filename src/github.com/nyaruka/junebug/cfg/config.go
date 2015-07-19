package cfg

import (
	"code.google.com/p/gcfg"
  "os"
  "errors"
  "fmt"
)

// Defines our configuration file format, this is all in the git/init format
type ConfigFormat struct {
	Directories struct {
		Connections string
		Outbox string
    Sent string
    Inbox string
    Handled string
	}
  Server struct {
    Port int
  }
}

var Config ConfigFormat

func GetSampleConfig() string {
  return "[directories]\n" +
         "connections = \"/usr/local/junebug/connections\"\n" +
         "inbox = \"/usr/local/junebug/inbox\"\n" +
         "outbox = \"/usr/local/junebug/outbox\"\n" +
         "sent = \"/usr/local/junebug/sent\"\n" +
         "handled = \"/usr/local/junebug/handled\"\n" +
         "\n" +
         "[server]\n" +
         "port = 8000\n"
}


func validateDirectory(key string, path string) error {
  fileInfo, err := os.Stat(path)
  if err != nil || !fileInfo.IsDir(){
    return errors.New(fmt.Sprintf("`%s` directory does not exist: '%s', check configuration file", key, path))
  } else {
    return nil
  }
}

func ReadConfig(filename string) (ConfigFormat, error) {
  err := gcfg.ReadFileInto(&Config, filename)

  err = validateDirectory("connections", Config.Directories.Connections)
  if err != nil {
      return Config, err
  }

  err = validateDirectory("outbox", Config.Directories.Outbox)
  if err != nil {
      return Config, err
  }

  err = validateDirectory("sent", Config.Directories.Sent)
  if err != nil {
      return Config, err
  }

  err = validateDirectory("inbox", Config.Directories.Inbox)
  if err != nil {
      return Config, err
  }

  err = validateDirectory("handled", Config.Directories.Handled)
  if err != nil {
      return Config, err
  }

  return Config, err
}
