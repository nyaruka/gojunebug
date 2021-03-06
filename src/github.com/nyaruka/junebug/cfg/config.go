package cfg

import (
	"github.com/scalingdata/gcfg"
	"errors"
	"fmt"
	"os"
)

// Defines our configuration file format, this is all in the git/init format
type ConfigFormat struct {
	Db struct {
		Filename    string }
	Server struct {
		Port int }
	Twitter struct {
		Consumer_Key string
		Consumer_Secret string }}

var Config ConfigFormat

func GetSampleConfig() string {
	return "[db]\n" +
		"filename = \"/usr/local/junebug/junebug.db\"\n" +
		"\n" +
		"[server]\n" +
		"port = 8000\n" +
	    "\n" +
		"[twitter]\n" +
		"consumer-key = \"put-your-twitter-application-consumer-key-here\"\n" +
	    "consumer-secret = \"put-your-twitter-application-consumer-secret-here\"\n"
}

func validateDirectory(key string, path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil || !fileInfo.IsDir() {
		return errors.New(fmt.Sprintf("`%s` directory does not exist: '%s', check configuration file", key, path))
	} else {
		return nil
	}
}

func ReadConfig(filename string) (ConfigFormat, error) {
	err := gcfg.ReadFileInto(&Config, filename)
	return Config, err
}