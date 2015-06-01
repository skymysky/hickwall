// +build windows

package main

import (
	"code.google.com/p/winsvc/svc"
	"github.com/oliveagle/hickwall/command"
	// "github.com/oliveagle/hickwall/config"
	"github.com/oliveagle/hickwall/hickwall"
	//	"github.com/oliveagle/hickwall/servicelib"
	"github.com/oliveagle/hickwall/utils"
	log "github.com/oliveagle/seelog"
	// "runtime/debug"
	"os"
	// "runtime/pprof"
	// "strconv"
	//	"github.com/davecheney/profile"
	"time"
)

var pid int

func init() {
	pid = os.Getpid()
}

type serviceHandler struct{}

func runWithoutService() {
	var (
		args    = []string{}
		r       = make(chan svc.ChangeRequest)
		changes = make(chan svc.Status)
	)

	go func() {
		for {
			select {
			case <-changes:
			}
		}
	}()

	runAsPrimaryService(args, r, changes)
}

func runAsPrimaryService(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	log.Info("runAsPrimaryService")
	defer utils.Recover_and_log()

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	//http://localhost:6060/debug/pprof/
	utils.HttpPprofServe(6060)

	//	after := time.After(time.Duration(8) * time.Minute)
	// f, _ := os.Create("d:\\cpu-" + strconv.Itoa(pid) + ".pprof")
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	//	cfg := profile.Config{
	//		MemProfile:     true,
	//		ProfilePath:    "./pprofs/", // store profiles in current directory
	//		NoShutdownHook: true,        // do not hook SIGINT
	//	}
	//	p := profile.Start(&cfg)
	//
	//	defer p.Stop()

	// utils.StartCPUProfile()
	// defer utils.StopCPUProfile()

	// go func() {
	// 	for {
	// 		<-time.After(time.Second * time.Duration(15))
	// 		debug.FreeOSMemory()
	// 	}
	// }()

	err := hickwall.Start()
	if err != nil {
		log.Critical("Failed To Start hickwall: %v", err)
		return
	} else {
		defer hickwall.Stop()
	}

	log.Debug("service event handling loop started ")
	// major loop for signal processing.
loop:
	for {
		select {
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
	return runAsPrimaryService(args, r, changes)
}

func runService(isDebug bool) {
	defer utils.Recover_and_log()
	log.Debug("runService")

	err = svc.Run(command.PrimaryService.Name(), &serviceHandler{})
	if err != nil {
		log.Errorf("runService: failed: %v\r\n", err)
	}
}
