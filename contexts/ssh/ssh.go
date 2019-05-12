package main

import "github.com/ovh/venom"

// Ctx represents a Test Context
type Ctx struct {
	Host            string
	HostKeyChecking string
	Port            string
	User            string
	Password        string
	PrivateKeyFile  string
	RemoteUser      string
	Retries         string
	Timeout         int64
	SSHExecutable   string
	SSHArgs         string
}

func (e Ctx) Manifest() venom.VenomModuleManifest {
	return venom.VenomModuleManifest{
		Name:    "ssh",
		Type:    "context",
		Version: venom.Version,
	}
}

// Run execute TestStep of type exec
func (e Ctx) Run(ctx venom.TestContext, step venom.TestStep) (venom.ExecutorResult, error) {
	return nil, nil
}

// Start the ctx
func (e Ctx) Start(ctx venom.TestContext) error {
	return nil
}
