package venom

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_executorModule(t *testing.T) {
	m := executorModule{
		entrypoint: "dist/executors/http/http_" + runtime.GOOS + "_" + runtime.GOARCH,
	}

	v := New()
	v.init()
	v.LogLevel = LogLevelDebug
	l := TestLogger{t}

	ctxMod, _ := v.getContextModule(nil)
	ctx, _ := ctxMod.New(context.Background(), nil, v, l)

	executor, err := m.New(ctx, v, l)
	assert.NoError(t, err)
	assert.NotNil(t, executor)

	res, err := executor.Run(ctx, nil)
	assert.NoError(t, err)
	assert.Nil(t, res)

}

func Test_getExecutorModule(t *testing.T) {
	v := New()
	v.init()
	v.ConfigurationDirectory = "./dist"
	v.LogLevel = LogLevelDebug

	step := TestStep{
		"type": "http",
	}

	mod, err := v.getExecutorModule(step)
	assert.NoError(t, err)
	assert.NotNil(t, mod)

	step = TestStep{
		"type": "notfound",
	}

	mod, err = v.getExecutorModule(step)
	assert.Error(t, err)
	assert.Nil(t, mod)

}
