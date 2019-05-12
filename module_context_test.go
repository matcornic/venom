package venom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getContextModule(t *testing.T) {
	v := New()
	v.init()
	v.ConfigurationDirectory = "./dist"
	v.LogLevel = LogLevelDebug

	ctxData := HH{
		"type": "ssh",
	}

	mod, err := v.getContextModule(ctxData)
	assert.NoError(t, err)
	assert.NotNil(t, mod)

	ctxData = HH{
		"type": "notfound",
	}

	mod, err = v.getContextModule(ctxData)
	assert.Error(t, err)
	assert.Nil(t, mod)

}
