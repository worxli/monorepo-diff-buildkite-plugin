package plugin_test

import (
	"io/ioutil"
	"os"
	"testing"

	plg "github.com/chronotc/monorepo-diff-buildkite-plugin/plugin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// disable logs in test
	log.SetOutput(ioutil.Discard)

	// set some env variables for using in tests
	os.Setenv("BUILDKITE_COMMIT", "123")
	os.Setenv("BUILDKITE_MESSAGE", "fix: temp file not correctly deleted")
	os.Setenv("BUILDKITE_BRANCH", "go-rewrite")
	os.Setenv("env3", "env-3")
	os.Setenv("env4", "env-4")
	os.Setenv("TEST_MODE", "true")

	run := m.Run()

	os.Exit(run)
}

func TestUploadPipelineCallsBuildkiteAgentCommand(t *testing.T) {
	plugin := plg.Plugin{Diff: "echo ./foo-service"}
	args, err := plugin.UploadPipeline("pipeline.txt")

	assert.Equal(t, []string{"pipeline", "upload", "pipeline.txt"}, args)
	assert.EqualError(t, err, "command `buildkite-agent` failed: exec: \"buildkite-agent\": executable file not found in $PATH")
}

func TestUploadPipelineCallsBuildkiteAgentCommandWithInterpolation(t *testing.T) {
	plugin := plg.Plugin{Diff: "echo ./foo-service", Interpolation: true}
	args, err := plugin.UploadPipeline("pipeline.txt")

	assert.Equal(t, []string{"pipeline", "upload", "pipeline.txt", "--no-interpolation"}, args)
	assert.EqualError(t, err, "command `buildkite-agent` failed: exec: \"buildkite-agent\": executable file not found in $PATH")
}

func TestDiff(t *testing.T) {
	want := []string{
		"services/foo/serverless.yml",
		"services/bar/config.yml",
		"ops/bar/config.yml",
		"README.md",
	}

	got, err := plg.Diff(`echo services/foo/serverless.yml
services/bar/config.yml

ops/bar/config.yml
README.md`)

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPipelinesToTriggerGetsListOfPipelines(t *testing.T) {
	want := []string{"service-1", "service-2", "service-4"}

	watch := []plg.WatchConfig{
		{
			Paths: []string{"watch-path-1"},
			Step:  plg.Step{Trigger: "service-1"},
		},
		{
			Paths: []string{"watch-path-2/", "watch-path-3/", "watch-path-4"},
			Step:  plg.Step{Trigger: "service-2"},
		},
		{
			Paths: []string{"watch-path-5"},
			Step:  plg.Step{Trigger: "service-3"},
		},
		{
			Paths: []string{"watch-path-2"},
			Step:  plg.Step{Trigger: "service-4"},
		},
	}

	changedFiles := []string{
		"watch-path-1/text.txt",
		"watch-path-2/.gitignore",
		"watch-path-2/src/index.go",
		"watch-path-4/test/index_test.go",
	}

	pipelines, err := plg.StepsToTrigger(changedFiles, watch)
	assert.NoError(t, err)
	var got []string

	for _, v := range pipelines {
		got = append(got, v.Trigger)
	}

	assert.Equal(t, want, got)
}

func TestPipelinesStepsToTrigger(t *testing.T) {

	testCases := map[string]struct {
		ChangedFiles []string
		WatchConfigs []plg.WatchConfig
		Expected     []plg.Step
	}{
		"service-1": {
			ChangedFiles: []string{
				"watch-path-1/text.txt",
				"watch-path-2/.gitignore",
			},
			WatchConfigs: []plg.WatchConfig{{
				Paths: []string{"watch-path-1"},
				Step:  plg.Step{Trigger: "service-1"},
			}},
			Expected: []plg.Step{
				{Trigger: "service-1"},
			},
		},
		"service-1-2": {
			ChangedFiles: []string{
				"watch-path-1/text.txt",
				"watch-path-2/.gitignore",
			},
			WatchConfigs: []plg.WatchConfig{
				{
					Paths: []string{"watch-path-1"},
					Step:  plg.Step{Trigger: "service-1"},
				},
				{
					Paths: []string{"watch-path-2"},
					Step:  plg.Step{Trigger: "service-2"},
				},
			},
			Expected: []plg.Step{
				{Trigger: "service-1"},
				{Trigger: "service-2"},
			},
		},
		"extension wildcard": {
			ChangedFiles: []string{
				"text.txt",
				".gitignore",
			},
			WatchConfigs: []plg.WatchConfig{
				{
					Paths: []string{"*.txt"},
					Step:  plg.Step{Trigger: "txt"},
				},
			},
			Expected: []plg.Step{
				{Trigger: "txt"},
			},
		},
		"extension wildcard in subdir": {
			ChangedFiles: []string{
				"docs/text.txt",
			},
			WatchConfigs: []plg.WatchConfig{
				{
					Paths: []string{"docs/*.txt"},
					Step:  plg.Step{Trigger: "txt"},
				},
			},
			Expected: []plg.Step{
				{Trigger: "txt"},
			},
		},
		"directory wildcard": {
			ChangedFiles: []string{
				"docs/text.txt",
			},
			WatchConfigs: []plg.WatchConfig{
				{
					Paths: []string{"**/text.txt"},
					Step:  plg.Step{Trigger: "txt"},
				},
			},
			Expected: []plg.Step{
				{Trigger: "txt"},
			},
		},
		"directory and extension wildcard": {
			ChangedFiles: []string{
				"package/other.txt",
			},
			WatchConfigs: []plg.WatchConfig{
				{
					Paths: []string{"*/*.txt"},
					Step:  plg.Step{Trigger: "txt"},
				},
			},
			Expected: []plg.Step{
				{Trigger: "txt"},
			},
		},
		"double directory and extension wildcard": {
			ChangedFiles: []string{
				"package/docs/other.txt",
			},
			WatchConfigs: []plg.WatchConfig{
				{
					Paths: []string{"**/*.txt"},
					Step:  plg.Step{Trigger: "txt"},
				},
			},
			Expected: []plg.Step{
				{Trigger: "txt"},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			steps, err := plg.StepsToTrigger(tc.ChangedFiles, tc.WatchConfigs)
			assert.NoError(t, err)
			assert.Equal(t, tc.Expected, steps)
		})
	}
}

func TestDetermineSteps(t *testing.T) {
	want := []plg.Step{
		{
			Trigger: "foo-service-pipeline",
			Build:   plg.Build{Message: "build message"},
		},
	}

	plugin := plg.Plugin{
		Diff: "echo foo-service/config.yaml",
		Watch: []plg.WatchConfig{
			{
				Paths: []string{
					"foo-service/",
				},
				Step: plg.Step{
					Trigger: "foo-service-pipeline",
					Build:   plg.Build{Message: "build message"},
				},
			},
		},
	}

	steps, err := plugin.DetermineSteps()

	assert.NoError(t, err)
	assert.Equal(t, want, steps)
}

func TestGeneratePipeline(t *testing.T) {
	steps := []plg.Step{
		{
			Trigger: "foo-service-pipeline",
			Build:   plg.Build{Message: "build message"},
		},
	}

	want :=
		`steps:
- trigger: foo-service-pipeline
  build:
    message: build message
- wait
- command: echo "hello world"
- command: cat ./file.txt`

	plugin := plg.Plugin{
		Wait: true,
		Hooks: []plg.HookConfig{
			{Command: "echo \"hello world\""},
			{Command: "cat ./file.txt"},
		},
	}

	pipeline, err := plugin.GeneratePipeline(steps)

	assert.NoError(t, err)
	assert.Equal(t, []byte(want), pipeline)
}
