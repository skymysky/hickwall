// this is a wrapper of two opensource project to implement
// a unanimous interface to install, remove, start, stop services

package servicelib

import (
	"fmt"
	"github.com/oliveagle/daemon"
	"github.com/oliveagle/hickwall/config"
	"os"
	"path/filepath"
)

type IService interface {
	InstallService() error
	RemoveService() error
	Status() (State, error)
	StartService(args ...string) error
	StopService() error
	PauseService() error
	ContinueService() error
	Version() error
	Name() string
}

type State uint

const (
	Unknown = iota
	Stopped
	StartPending
	StopPending
	Running
	ContinuePending
	PausePending
	Paused
)

func StateToString(s State) string {
	switch s {
	case Stopped:
		return "Stopped"
	case StartPending:
		return "StartPending"
	case StopPending:
		return "StopPending"
	case Running:
		return "Running"
	case ContinuePending:
		return "ContinuePending"
	case PausePending:
		return "PausePending"
	case Paused:
		return "Paused"
	default:
		return "Unknown"
	}
}

func HandleCmd(isrv IService, cmd string) (err error) {
	switch cmd {
	case "install":
		err = isrv.InstallService()
	case "remove":
		err = isrv.RemoveService()
	case "start":
		err = isrv.StartService()
	case "stop":
		err = isrv.StopService()
	case "pause":
		err = isrv.PauseService()
	case "continue":
		err = isrv.ContinueService()
	case "status":
		_, err = isrv.Status()
	case "version":
		err = isrv.Version()
	default:
		err = fmt.Errorf("invalid command %s", cmd)
	}
	return
}

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

type Service struct {
	daemon.Daemon
	name string
	desc string
}

func NewService(name, desc string) *Service {
	srv, err := daemon.New(name, desc)
	if err != nil {
		fmt.Println("Error: cannot create daemon Service: ", err)
		os.Exit(1)
	}
	return &Service{srv, name, desc}
}

func (this *Service) Version() (err error) {
	fmt.Println("Version: ", config.VERSION)
	return err
}

func (this *Service) Name() string {
	return this.name
}
