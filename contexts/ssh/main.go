package main

import (
	"github.com/ovh/venom/lib/cmd"
	"github.com/ovh/venom/lib/context"
)

func main() {
	var c Ctx
	if err := vcontext.Start(c); err != nil {
		cmd.ExitOnError(err)
	}
}
