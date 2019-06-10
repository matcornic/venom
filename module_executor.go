package venom

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/phayes/freeport"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/yaml.v2"
)

type executorModule struct {
	retry      int
	delay      int
	timeout    int
	entrypoint string
	manifest   VenomModuleManifest
}

func (e executorModule) Manifest() VenomModuleManifest {
	return e.manifest
}

func (e executorModule) New(ctx context.Context, v *Venom, l Logger) (Executor, error) {
	var es executorStarter
	es.starter.l = LoggerWithField(l, "executor", e.manifest.Name)
	es.starter.v = v
	es.executorModule = e
	es.starter.logServer = syslog.NewServer()
	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	es.starter.logServerAddress = "0.0.0.0:" + strconv.Itoa(port)
	es.starter.logServer.SetHandler(es.starter.logsHandler(ctx))
	es.starter.logServer.SetFormat(syslog.Automatic)
	l.Debugf("Starting syslog server on %s", es.starter.logServerAddress)
	if err := es.starter.logServer.ListenUDP(es.starter.logServerAddress); err != nil {
		return nil, err
	}
	if err := es.starter.logServer.ListenTCP(es.starter.logServerAddress); err != nil {
		return nil, err
	}
	go func(s *syslog.Server) {
		s.Boot()
		s.Wait()
	}(es.starter.logServer)

	go func(s *syslog.Server) {
		<-ctx.Done()
		s.Kill()
	}(es.starter.logServer)

	return &es, nil
}

func (e executorModule) GetDefaultAssertions(ctx TestContext) (*StepAssertions, error) {
	// Instanciate the execute command
	cmd := exec.CommandContext(ctx, e.entrypoint, "assertions")

	output := new(bytes.Buffer)
	cmd.Stdout = output
	cmd.Stderr = output
	// TODO: start the command in the right working directory
	cmd.Dir = ctx.GetWorkingDirectory()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start command: %v", err)
	}

	// Run the command and wait for the result
	waitErr := cmd.Wait()

	btes := output.Bytes()

	// Check error
	if waitErr != nil {
		return nil, fmt.Errorf("error: %v", waitErr)
	}

	if strings.TrimSpace(string(btes)) == "" {
		return nil, nil
	}

	// Unmarshal the result
	var res StepAssertions
	if err := yaml.Unmarshal(btes, &res); err != nil {
		return nil, fmt.Errorf("unable to parse module output: %v", err)
	}
	return &res, nil
}

type starter struct {
	v                *Venom
	l                Logger
	logServer        *syslog.Server
	logServerAddress string
}

type executorStarter struct {
	starter starter
	executorModule
}

func (e *executorStarter) Run(ctx TestContext, step TestStep) (ExecutorResult, error) {
	if step == nil {
		return nil, nil
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Instanciate the execute command
	cmd := exec.CommandContext(ctx, e.entrypoint, "execute", "--logger", e.starter.logServerAddress, "--log-level", e.starter.v.LogLevel)
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
		return nil, fmt.Errorf("unable to open stdin: %v", err)
	}

	encoder := yaml.NewEncoder(stdin)
	if err := encoder.Encode(step); err != nil {
		e.starter.l.Errorf("%#v", step)
		return nil, fmt.Errorf("unable to write to stdin: %v", err)
	}

	if err := stdin.Close(); err != nil {
		return nil, fmt.Errorf("unable to close stdin: %v", err)
	}

	output := new(bytes.Buffer)
	cmd.Stdout = output
	cmd.Stderr = output

	e.starter.l.Infof("Executing command %s", cmd.Path)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start command: %v", err)
	}

	// Run the command and wait for the result
	waitErr := cmd.Wait()

	btes := output.Bytes()

	// Check error
	if waitErr != nil {
		return nil, fmt.Errorf("error: %v", waitErr)
	}

	// Unmarshal the result
	var res ExecutorResult
	if err := yaml.Unmarshal(btes, &res); err != nil {
		return nil, fmt.Errorf("unable to parse module output: %v", err)
	}

	return res, nil
}

var (
	levelRegexp = regexp.MustCompile(`level=([a-z]*)`)
	msgRegexp   = regexp.MustCompile(`msg=(".*"|\w)`)
)

func (e *starter) logsHandler(ctx context.Context) syslog.Handler {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case logParts := <-channel:
				content, has := logParts["content"]
				if !has {
					continue
				}
				scontent, ok := content.(string)
				if !ok {
					continue
				}

				levelMatch := levelRegexp.FindStringSubmatch(scontent)
				if len(levelMatch) != 2 {
					continue
				}
				level := levelMatch[1]

				msgMatch := msgRegexp.FindStringSubmatch(scontent)
				if len(msgMatch) != 2 {
					continue
				}
				msg := msgMatch[1]

				msg = strings.TrimPrefix(msg, "\"")
				msg = strings.TrimSuffix(msg, "\"")

				msg = colorExecutor(msg)

				switch level {
				case "debug":
					e.l.Debugf(msg)
				case "info":
					e.l.Infof(msg)
				case "warning":
					e.l.Warningf(msg)
				case "error":
					e.l.Errorf(msg)
				case "fatal":
					e.l.Fatalf(msg)
				default:
					log.Println(level, msg)
				}
			}
		}
	}()
	return handler
}

func (v *Venom) getExecutorModule(step TestStep) (*executorModule, error) {
	allModules, err := v.ListModules()
	if err != nil {
		return nil, err
	}
	var mod *executorModule
	for _, m := range allModules {
		var manifest = m.Manifest()
		e, ok := m.(executorModule)
		var stepType = step.GetType()
		if stepType == "" {
			stepType = "exec"
		}
		if ok && manifest.Type == "executor" && stepType == manifest.Name {
			mod = &e
			break
		}
	}
	if mod == nil {
		return nil, fmt.Errorf("unrecognized type %s", step.GetType())
	}
	return mod, nil
}
