package logger

import (
	"github.com/op/go-logging"
	"os"
)

var Logger = logging.MustGetLogger("server")

var format = logging.MustStringFormatter(
	`%{color}%{time:2006-01-02 15:04:05} %{shortfile} %{longfunc}: %{level:.4s} %{message}`,
)

func Init() {
	console := logging.NewLogBackend(os.Stderr, "", 0)
	consoleFormatter := logging.NewBackendFormatter(console, format)
	consoleLeveled := logging.AddModuleLevel(consoleFormatter)
	consoleLeveled.SetLevel(logging.DEBUG, "")
	logging.SetBackend(consoleLeveled)
}
