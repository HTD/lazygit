//go:build !windows
// +build !windows

package gui

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/creack/pty"
	"github.com/jesseduffield/lazygit/pkg/integration"
	"github.com/stretchr/testify/assert"
)

// This file is quite similar to integration/main.go. The main difference is that this file is
// run via `go test` whereas the other is run via `test/lazyintegration/main.go` which provides
//  a convenient gui wrapper around our integration tests. The `go test` approach is better
// for CI and for running locally in the background to ensure you haven't broken
// anything while making changes. If you want to visually see what's happening when a test is run,
// you'll need to take the other approach
//
// As for this file, to run an integration test, e.g. for test 'commit', go:
// go test pkg/gui/gui_test.go -run /commit
//
// To update a snapshot for an integration test, pass UPDATE_SNAPSHOTS=true
// UPDATE_SNAPSHOTS=true go test pkg/gui/gui_test.go -run /commit
//
// integration tests are run in test/integration/<test_name>/actual and the final test does
// not clean up that directory so you can cd into it to see for yourself what
// happened when a test fails.
//
// To override speed, pass e.g. `SPEED=1` as an env var. Otherwise we start each test
// at a high speed and then drop down to lower speeds upon each failure until finally
// trying at the original playback speed (speed 1). A speed of 2 represents twice the
// original playback speed. Speed may be a decimal.

func Test(t *testing.T) {
	record := false
	updateSnapshots := os.Getenv("UPDATE_SNAPSHOTS") != ""
	speedEnv := os.Getenv("SPEED")
	includeSkipped := os.Getenv("INCLUDE_SKIPPED") != ""

	err := integration.RunTests(
		t.Logf,
		runCmdHeadless,
		func(test *integration.Test, f func(*testing.T) error) {
			t.Run(test.Name, func(t *testing.T) {
				err := f(t)
				assert.NoError(t, err)
			})
		},
		updateSnapshots,
		record,
		speedEnv,
		func(t *testing.T, expected string, actual string) {
			assert.Equal(t, expected, actual, fmt.Sprintf("expected:\n%s\nactual:\n%s\n", expected, actual))
		},
		includeSkipped,
	)

	assert.NoError(t, err)
}

func runCmdHeadless(cmd *exec.Cmd) error {
	cmd.Env = append(
		cmd.Env,
		"HEADLESS=true",
		"TERM=xterm",
	)

	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 100, Cols: 100})
	if err != nil {
		return err
	}

	_, _ = io.Copy(ioutil.Discard, f)

	return f.Close()
}

func TestGuiGenerateMenuCandidates(t *testing.T) {
	type scenario struct {
		testName    string
		cmdOut      string
		filter      string
		valueFormat string
		labelFormat string
		test        func([]commandMenuEntry, error)
	}

	scenarios := []scenario{
		{
			"Extract remote branch name",
			"upstream/pr-1",
			"(?P<remote>[a-z_]+)/(?P<branch>.*)",
			"{{ .branch }}",
			"Remote: {{ .remote }}",
			func(actualEntry []commandMenuEntry, err error) {
				assert.NoError(t, err)
				assert.EqualValues(t, "pr-1", actualEntry[0].value)
				assert.EqualValues(t, "Remote: upstream", actualEntry[0].label)
			},
		},
		{
			"Multiple named groups with empty labelFormat",
			"upstream/pr-1",
			"(?P<remote>[a-z]*)/(?P<branch>.*)",
			"{{ .branch }}|{{ .remote }}",
			"",
			func(actualEntry []commandMenuEntry, err error) {
				assert.NoError(t, err)
				assert.EqualValues(t, "pr-1|upstream", actualEntry[0].value)
				assert.EqualValues(t, "pr-1|upstream", actualEntry[0].label)
			},
		},
		{
			"Multiple named groups with group ids",
			"upstream/pr-1",
			"(?P<remote>[a-z]*)/(?P<branch>.*)",
			"{{ .group_2 }}|{{ .group_1 }}",
			"Remote: {{ .group_1 }}",
			func(actualEntry []commandMenuEntry, err error) {
				assert.NoError(t, err)
				assert.EqualValues(t, "pr-1|upstream", actualEntry[0].value)
				assert.EqualValues(t, "Remote: upstream", actualEntry[0].label)
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.testName, func(t *testing.T) {
			s.test(NewDummyGui().GenerateMenuCandidates(s.cmdOut, s.filter, s.valueFormat, s.labelFormat))
		})
	}
}
