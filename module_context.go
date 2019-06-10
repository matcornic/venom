package venom

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/phayes/freeport"

	"github.com/spf13/cast"
	"gopkg.in/mcuadros/go-syslog.v2"
	yaml "gopkg.in/yaml.v2"
)

func (v *Venom) getContextModule(ctxData HH) (*contextModule, error) {
	if ctxData == nil {
		ctxData = HH{}
	}
	s := cast.ToString(ctxData["type"])
	if s == "" || s == "default" {
		return &contextModule{
			manifest: VenomModuleManifest{
				Name:        "default",
				Type:        "context",
				Version:     Version,
				Description: "venom default context",
			},
		}, nil
	}

	allModules, err := v.ListModules()
	if err != nil {
		return nil, err
	}
	var mod *contextModule
	for _, m := range allModules {
		var manifest = m.Manifest()
		e, ok := m.(contextModule)
		if ok && manifest.Type == "context" && s == manifest.Name {
			mod = &e
			break
		}
	}
	if mod == nil {
		return nil, fmt.Errorf("unrecognized type %s", s)
	}
	return mod, nil
}

type contextModule struct {
	entrypoint string
	manifest   VenomModuleManifest
	commonContext
}

func (c contextModule) Manifest() VenomModuleManifest {
	return c.manifest
}

func (e contextModule) New(ctx context.Context, values HH, v *Venom, l Logger) (TestContext, error) {
	if e.manifest.Name == "default" {
		return &commonContext{ctx, values, ""}, nil
	}

	var cs contextStarter
	cs.starter.l = LoggerWithField(l, "executor", e.manifest.Name)
	cs.starter.v = v
	cs.contextModule = e
	cs.commonContext.Context = ctx
	cs.commonContext.values = values
	cs.starter.logServer = syslog.NewServer()
	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	cs.starter.logServerAddress = "0.0.0.0:" + strconv.Itoa(port)
	cs.starter.logServer.SetHandler(cs.starter.logsHandler(ctx))
	cs.starter.logServer.SetFormat(syslog.Automatic)
	l.Debugf("Starting syslog server on %s", cs.starter.logServerAddress)
	if err := cs.starter.logServer.ListenUDP(cs.starter.logServerAddress); err != nil {
		return nil, err
	}
	if err := cs.starter.logServer.ListenTCP(cs.starter.logServerAddress); err != nil {
		return nil, err
	}
	go func(s *syslog.Server) {
		s.Boot()
		s.Wait()
	}(cs.starter.logServer)

	go func(s *syslog.Server) {
		<-ctx.Done()
		s.Kill()
	}(cs.starter.logServer)

	return &cs, nil
}

type commonContext struct {
	context.Context
	values           HH
	workingDirectory string
}

func (e *commonContext) SetWorkingDirectory(s string) {
	e.workingDirectory = s
}
func (e *commonContext) GetWorkingDirectory() string {
	return e.workingDirectory
}

func (e *commonContext) Bag() HH {
	return e.values
}

type contextStarter struct {
	starter starter
	contextModule
}

func (e *contextStarter) Start(ctx TestContext) error {
	if ctx == nil {
		return nil
	}

	// Instanciate the execute command
	cmd := exec.CommandContext(ctx, e.entrypoint, "start", "--logger", e.starter.logServerAddress, "--log-level", e.starter.v.LogLevel)
	cmd.Dir = ctx.GetWorkingDirectory()

	bag := ctx.Bag()
	cmd.Env = os.Environ()
	for k := range bag {
		v := bag.Get(k)
		e.starter.l.Debugf("Setting context value: %s:%s", k, v)
		cmd.Env = append(cmd.Env, "VENOM_CTX_"+k+"="+v)
	}

	// Write in the stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("unable to open stdin: %v", err)
	}

	encoder := yaml.NewEncoder(stdin)
	if err := encoder.Encode(ctx); err != nil {
		e.starter.l.Errorf("%#v", ctx)
		return fmt.Errorf("unable to write to stdin: %v", err)
	}

	if err := stdin.Close(); err != nil {
		return fmt.Errorf("unable to close stdin: %v", err)
	}

	output := new(bytes.Buffer)
	cmd.Stdout = output
	cmd.Stderr = output

	e.starter.l.Infof("Executing command %s", cmd.Path)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start command: %v", err)
	}

	// Read the start output

	return nil
}
