package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCreateTmpFile(t *testing.T) {
	tmp := createTmpFile([]byte("data"))
	defer os.Remove(tmp.Name())

	got, _ := ioutil.ReadFile(tmp.Name())
	assert.Equal(t, "data", string(got))
}

func TestSetupLogger(t *testing.T) {
	setupLogger("debug")
	assert.Equal(t, log.GetLevel(), logrus.DebugLevel)
	setupLogger("weird level")
	assert.Equal(t, log.GetLevel(), logrus.InfoLevel)
}
