package venom

import (
	"context"
	"fmt"

	"github.com/spf13/cast"

	"github.com/pkg/errors"
)

type contextModule struct {
	entrypoint string
	manifest   VenomModuleManifest
}

func (c contextModule) Manifest() VenomModuleManifest {
	return c.manifest
}

func (e contextModule) New(ctx context.Context, values HH, v *Venom, l Logger) (TestContext, error) {
	if e.manifest.Name == "default" {
		return &commonContext{ctx, values, ""}, nil
	}
	return nil, fmt.Errorf("unrecognized context %s", e.manifest.Name)
}

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
	return nil, errors.New("unsupported context")
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
