package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/bmatcuk/doublestar/v2"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Pipeline is Buildkite pipeline definition
type Pipeline struct {
	Steps []Step
}

func (p Plugin) UploadPipeline(pipeline string) ([]string, error) {
	args := []string{"pipeline", "upload", pipeline}
	if p.Interpolation {
		args = append(args, "--no-interpolation")
	}

	_, err := executeCommand("buildkite-agent", args)
	if err != nil {
		return args, err
	}

	return args, nil
}

func Diff(command string) ([]string, error) {
	log.Infof("Running diff command: %s", command)

	split := strings.Split(command, " ")
	cmd, args := split[0], split[1:]

	output, err := executeCommand(cmd, args)
	if err != nil {
		return nil, fmt.Errorf("diff failed: %v", err)
	}

	f := func(c rune) bool {
		return c == '\n'
	}

	return strings.FieldsFunc(strings.TrimSpace(output), f), nil
}

func StepsToTrigger(files []string, watch []WatchConfig) ([]Step, error) {
	steps := []Step{}

	for _, w := range watch {
		for _, p := range w.Paths {
			for _, f := range files {
				match, err := matchPath(p, f)
				if err != nil {
					return nil, err
				}
				if match {
					steps = append(steps, w.Step)
					break
				}
			}
		}
	}

	return dedupSteps(steps), nil
}

// matchPath checks if the file f matches the path p.
func matchPath(p string, f string) (bool, error) {
	// If the path contains a glob, the `path.Match`
	// method is used to determine the match.
	if strings.Contains(p, "*") {
		match, err := doublestar.Match(p, f)
		if err != nil {
			return false, fmt.Errorf("path matching failed: %v", err)
		}
		if match {
			return true, nil
		}
	}
	if strings.HasPrefix(f, p) {
		return true, nil
	}
	return false, nil
}

func dedupSteps(steps []Step) []Step {
	unique := []Step{}
	for _, p := range steps {
		duplicate := false
		for _, t := range unique {
			if reflect.DeepEqual(p, t) {
				duplicate = true
				break
			}
		}

		if !duplicate {
			unique = append(unique, p)
		}
	}

	return unique
}

func (p Plugin) DetermineSteps() ([]Step, error) {
	diffOutput, err := Diff(p.Diff)
	if err != nil {
		return nil, err
	}

	if len(diffOutput) < 1 {
		return nil, nil
	}

	log.Debug("Output from diff: \n" + strings.Join(diffOutput, "\n"))

	steps, err := StepsToTrigger(diffOutput, p.Watch)
	if err != nil {
		return nil, err
	}
	return steps, nil
}

func (p Plugin) GeneratePipeline(steps []Step) ([]byte, error) {
	pipeline := Pipeline{Steps: steps}
	data, err := yaml.Marshal(&pipeline)
	if err != nil {
		return nil, errors.New("could not serialize the pipeline")
	}

	if p.Wait {
		data = append(data, "- wait"...)
	}

	for _, cmd := range p.Hooks {
		data = append(data, "\n- command: "+cmd.Command...)
	}

	// Disable logging in context of go tests.
	if env("TEST_MODE", "") != "true" {
		fmt.Printf("Generated Pipeline:\n%s\n", string(data))
	}

	return data, nil
}
