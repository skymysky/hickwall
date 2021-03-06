package command

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/oliveagle/hickwall/servicelib"
	// "os"
	// "sync"
	"github.com/oliveagle/hickwall/config"
	"github.com/oliveagle/hickwall/logging"
)

var PrimaryService = servicelib.NewService("hickwall", "monitoring system")

func CmdShowConfig(c *cli.Context) {
	logging.Debug("CmdShowConfig")

	fmt.Printf("CoreConfig: %+v\n", config.CoreConf)

	//TODO: get runtime config from running core.
}
