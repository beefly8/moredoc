package daemore

import (
	"errors"
	"os"

	"github.com/kardianos/service"
)

type Daemon struct {
	config   *service.Config
	errs     chan error
	callback func()
	Service  service.Service
}

type DaemonOption struct {
	Name        string
	DisplayName string
	Description string
	Callback    func()
}

func NewDaemon(opt DaemonOption, username ...string) (d *Daemon, err error) {
	config := &service.Config{
		Name:        opt.Name,
		DisplayName: opt.DisplayName,
		Description: opt.Description,
	}
	if len(username) > 0 {
		config.UserName = username[0]
	}

	d = &Daemon{
		config:   config,
		errs:     make(chan error, 100),
		callback: opt.Callback,
	}

	if opt.Callback == nil {
		err = errors.New("callback is required")
		return
	}

	d.Service, err = service.New(d, d.config)
	if err != nil {
		return nil, err
	}
	return
}

func (d *Daemon) Config() *service.Config {
	return d.config
}

func (d *Daemon) Start(s service.Service) error {
	go d.Run()
	return nil
}

func (d *Daemon) Run() {
	if d.callback == nil {
		return
	}
	d.callback()
}

func (d *Daemon) Stop(s service.Service) error {
	if service.Interactive() {
		os.Exit(0)
	}
	return nil
}

func (d *Daemon) ServiceInstall(args ...string) (err error) {
	d.config.Arguments = args
	s, err := service.New(d, d.config)
	if err != nil {
		return err
	}
	return s.Install()
}

func (d *Daemon) ServiceUninstall() (err error) {
	s, err := service.New(d, d.config)
	if err != nil {
		return err
	}
	return s.Uninstall()
}

func (d *Daemon) ServiceRestart() (err error) {
	s, err := service.New(d, d.config)
	if err != nil {
		return err
	}
	s.Stop()
	return s.Start()
}

func (d *Daemon) ServiceStop() (err error) {
	s, err := service.New(d, d.config)
	if err != nil {
		return err
	}
	return s.Stop()
}

func (d *Daemon) ServiceStart() (err error) {
	s, err := service.New(d, d.config)
	if err != nil {
		return err
	}
	return s.Start()
}

func (d *Daemon) ServiceStatus() (status service.Status, err error) {
	s, err := service.New(d, d.config)
	if err != nil {
		return status, err
	}
	return s.Status()
}
