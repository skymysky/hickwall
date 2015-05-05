// +build windows

package main

import (
	"code.google.com/p/winsvc/svc"
	// "fmt"
	"github.com/oliveagle/hickwall/backends"
	"github.com/oliveagle/hickwall/collectorlib"
	"github.com/oliveagle/hickwall/collectors"
	"github.com/oliveagle/hickwall/command"
	"github.com/oliveagle/hickwall/servicelib"
	"github.com/oliveagle/hickwall/utils"
	log "github.com/oliveagle/seelog"
	"time"
)

func start_service_if_stopped(service *servicelib.Service) {
	state, err := service.Status()
	if err != nil {
		log.Errorf("CmdServiceStatus Error: %v", err)
		return
	}
	if state == servicelib.Stopped {
		log.Warnf("service %s is stopped! trying to start service again", service.Name())

		err := service.StartService()
		if err != nil {
			log.Error("start service failed: ", err)
		} else {
			log.Info("service %s started", service.Name())
		}
	}
}

type serviceHandler struct{}

func runAsPrimaryService(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	mdCh := make(chan collectorlib.MultiDataPoint)

	log.Info("runAsPrimaryService")
	defer log.Flush()

	collectors.RunBuiltinCollectors(mdCh)
	log.Debug("all builtin collectors turned on")

	utils.HttpPprofServe(6060)

	go LoadConfigAndReload(mdCh)

	// where to watch config and reload
	// go func() {
	// 	for resp := range config.WatchConfig() {
	// 		if resp.Err != nil {
	// 			log.Critical("watch config error: %v", resp.Err)
	// 		} else {
	// 			defer log.Flush()

	// 			log.Debug("new config is comming")

	// 			collectors.StopCustomizedCollectors()
	// 			collectors.RemoveAllCustomizedCollectors()

	// 			log.Debug("Stopped Customized Collectors and Removed them")

	// 			backends.CloseBackends()
	// 			backends.RemoveAllBackends()

	// 			log.Debug("Stopped backends and removed them")

	// 			config.UpdateRuntimeConf(resp.Config)

	// 			log.Debug("Updated Runtime Conf with the new one")

	// 			collectors.CreateCustomizedCollectorsFromRuntimeConf()
	// 			log.Debug("Created Customized Colletors")

	// 			backends.CreateBackendsFromRuntimeConf()
	// 			log.Debug("Created backends")

	// 			collectors.RunCustomizedCollectors(mdCh)
	// 			log.Debug("all customized collectors turned on")

	// 			backends.RunBackends()
	// 			log.Debug("all backends turned on")

	// 			log.Debug("new config applied")
	// 		}
	// 	}
	// }()

	// major loop for signal processing.
loop:
	for {
		select {
		case md, _ := <-mdCh:
			for _, p := range md {
				log.Trace(" point ---> ", p)
			}
			backends.WriteToBackends(md)
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Info("svc.Stop or svc.Shutdown is triggered")
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				log.Info("svc.Pause not implemented yet")
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				log.Info("svc.Continue not implemented yet")
			default:
				log.Errorf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	log.Info("serviceHandler stopped")
	return
}

func runAsHelperService(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// NOTE: helper service should not write log to file. otherwise, multiple process write to same log file will cause log
	// rotate have unexpected behaviors.

	log.Info("helper service started")

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	tick := time.Tick(time.Second * time.Duration(1))

	// major loop for signal processing.
loop:
	for {
		select {
		case <-tick:
			go start_service_if_stopped(command.PrimaryService)
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Info("svc.Stop or svc.Shutdown is triggered")
				break loop
			default:
				log.Error("unexpected control request #%d", c)

			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	log.Error("helper service stopped")
	return

}

func (this *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	defer log.Flush()

	log.Infof("serviceHandler.Execute: %v", args)

	// if len(args) > 0 {
	// 	svc_name := args[0]
	// 	if svc_name == "hickwall" {
	// 		return runAsPrimaryService(args, r, changes)
	// 	} else {
	// 		return runAsHelperService(args, r, changes)
	// 	}
	// }

	return runAsPrimaryService(args, r, changes)
}

func runService(isDebug bool) {
	defer log.Flush()
	log.Debug("runService")
	err = svc.Run(command.PrimaryService.Name(), &serviceHandler{})
	if err != nil {
		log.Errorf("runService: failed: %v\r\n", err)
	}
}
