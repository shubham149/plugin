// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/drone/plugin/plugin/internal/environ"
	"github.com/nektos/act/cmd"
	"github.com/pkg/errors"
)

// Execer executes a github action.
type Execer struct {
	Name    string
	Environ []string
	Stdout  io.Writer
	Stderr  io.Writer
}

// Exec executes a github action.
func (e *Execer) Exec(ctx context.Context) error {
	envVars := environ.Map(e.Environ)
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	workflowFile := fmt.Sprintf("%s/workflow.yml", tmpDir)

	if err := createWorkflowFile(workflowFile, e.Name, envVars); err != nil {
		return err
	}

	oldOsArgs := os.Args
	defer func() { os.Args = oldOsArgs }()

	os.Args = []string{
		"action",
		"-W",
		workflowFile,
		"-P",
		fmt.Sprintf("-self-hosted=-self-hosted"),
		"-b",
		"--detect-event",
	}

	if eventPayload, ok := envVars["PLUGIN_EVENT_PAYLOAD"]; ok {
		eventPayloadFile := fmt.Sprintf("%s/event.yml", tmpDir)

		if err := ioutil.WriteFile(eventPayloadFile, []byte(eventPayload), 0644); err != nil {
			return errors.Wrap(err, "failed to write event payload to file")
		}

		os.Args = append(os.Args, "--eventpath", eventPayloadFile)
	}

	cmd.Execute(ctx, "1.1")
	return nil
}

// trace writes each command to stdout with the command wrapped in an xml
// tag so that it can be extracted and displayed in the logs.
func trace(cmd *exec.Cmd) {
	fmt.Fprintf(os.Stdout, "+ %s\n", strings.Join(cmd.Args, " "))
}
