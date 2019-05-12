package vcontext

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
	"github.com/ovh/venom/lib/executor"
	yaml "gopkg.in/yaml.v2"
)

func getContextStartFunc(c wrappedCommon) func(vals cmd.Values) *cmd.Error {
	return func(vals cmd.Values) *cmd.Error {
		input, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return cmd.NewError(502, "unable to read stdin: %v", err)
		}

		var step venom.TestStep
		if err := yaml.Unmarshal(input, &step); err != nil {
			return cmd.NewError(502, "unable to parse stdin: %v", err)
		}

		loggerAddress := vals.GetString("logger")
		logLevel := vals.GetString("log-level")
		if err := executor.NewLogger(loggerAddress, logLevel); err != nil {
			return cmd.NewError(502, "logger error: %v", err)
		}

		t0 := time.Now()
		name := c.Manifest().Name
		executor.Debugf(name + ".Run> Begin")
		defer func() {
			executor.Debugf(name+".Run> End (%.3f seconds)", time.Since(t0).Seconds())
		}()

		if err := c.Start(executor.NewContextFromEnv()); err != nil {
			executor.Errorf("Error: %v", err)
			return cmd.NewError(502, "executor error: %v", err)
		}
		return nil
	}
}
