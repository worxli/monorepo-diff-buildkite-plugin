package main

import (
	"io/ioutil"
	"os"

	plg "github.com/chronotc/monorepo-diff-buildkite-plugin/plugin"
	log "github.com/sirupsen/logrus"
)

func setupLogger(logLevel string) {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	ll, err := log.ParseLevel(logLevel)

	if err != nil {
		log.Debugf("error parsing log level: %v", err)
		ll = log.InfoLevel
	}

	log.SetLevel(ll)
}

// Version of plugin
var Version string

func main() {
	log.Infof("--- :one: monorepo-diff %s", Version)

	plugins := ""
	if value, ok := os.LookupEnv("BUILDKITE_PLUGINS"); ok {
		plugins = value
	}

	plugin, err := plg.InitializePlugin(plugins)
	if err != nil {
		log.Fatal(err)
	}

	setupLogger(plugin.LogLevel)

	steps, err := plugin.DetermineSteps()
	if err != nil {
		log.Fatal(err)
	}
	if len(steps) == 0 {
		log.Info("No changes detected. Skipping pipeline upload.")
		return
	}

	pipeline, err := plugin.GeneratePipeline(steps)
	if err != nil {
		log.Fatal(err)
	}

	tmp := createTmpFile(pipeline)
	defer os.Remove(tmp.Name())

	_, err = plugin.UploadPipeline(tmp.Name())
	if err != nil {
		log.Fatal(err)
	}
}

func createTmpFile(data []byte) *os.File {
	tmp, err := ioutil.TempFile(os.TempDir(), "bmrd-")
	if err != nil {
		log.Fatal(err)
	}
	if err = ioutil.WriteFile(tmp.Name(), data, 0644); err != nil {
		log.Fatal(err)
	}
	return tmp
}
