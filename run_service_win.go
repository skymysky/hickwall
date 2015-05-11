// +build windows

package main

import (
	"code.google.com/p/winsvc/svc"
	// "fmt"
	"github.com/oliveagle/hickwall/backends"
	// "github.com/oliveagle/hickwall/collectorlib"
	"github.com/oliveagle/hickwall/collectors"
	"github.com/oliveagle/hickwall/command"
	"github.com/oliveagle/hickwall/config"
	"github.com/oliveagle/hickwall/servicelib"
	"github.com/oliveagle/hickwall/utils"
	log "github.com/oliveagle/seelog"
	"time"
)

func start_service_if_stopped(service *servicelib.Service) {
	defer utils.Recover_and_log()

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
	defer utils.Recover_and_log()
	log.Info("runAsPrimaryService")

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	//http://localhost:6060/debug/pprof/
	utils.HttpPprofServe(6060)

	go LoadConfigAndWatching()

	// go func() {
	// 	for {
	// 		<-time.After(time.Second * time.Duration(15))
	// 		ReloadWithRuntimeConfig()
	// 	}
	// }()

	// major loop for signal processing.
loop:
	for {
		select {
		case md, _ := <-collectors.GetDataChan():
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

func (this *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	defer utils.Recover_and_log()

	log.Infof("serviceHandler.Execute: %v", args)
	return runAsPrimaryService(args, r, changes)
}

func runService(isDebug bool) {
	defer utils.Recover_and_log()
	// panic("hahaah")

	if !config.IsCoreConfigLoaded() {
		log.Critical("core config not loaded.")
		return
	}

	log.Debug("runService")
	err = svc.Run(command.PrimaryService.Name(), &serviceHandler{})
	if err != nil {
		log.Errorf("runService: failed: %v\r\n", err)
	}
}
