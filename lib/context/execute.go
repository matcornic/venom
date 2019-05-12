package vcontext

import (
	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
)

func getContextExecuteFunc(c wrappedCommon) func(vals cmd.Values) *cmd.Error {
	return func(vals cmd.Values) *cmd.Error {
		_, ok := c.Common.(venom.Executor)
		if !ok {
			return nil
		}
		return nil
	}
}
