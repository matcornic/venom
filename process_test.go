package venom

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

func Test_Process(t *testing.T) {
	v := New()
	v.logger = logrus.New()
	v.logger.SetOutput(TestLogger{})
	v.ConfigurationDirectory = filepath.Join("dist")
	r, err := v.Process(context.Background(), []string{"tests/*.yml"})
	assert.NoError(t, err)
	if err == nil {
		assert.True(t, len(r.TestSuites) == len(v.testsuites), "not the right number of testsuites", len(r.TestSuites), len(v.testsuites))
		assert.True(t, r.Total >= len(v.testsuites), "total seems wrong", r.Total, len(v.testsuites))
	}
}
