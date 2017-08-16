/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/containerd/containerd"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri/v1alpha1/runtime"

	"github.com/kubernetes-incubator/cri-containerd/pkg/server/agents"
	containerstore "github.com/kubernetes-incubator/cri-containerd/pkg/store/container"
)

// StartContainer starts the container.
func (c *criContainerdService) StartContainer(ctx context.Context, r *runtime.StartContainerRequest) (retRes *runtime.StartContainerResponse, retErr error) {
	glog.V(2).Infof("StartContainer for %q", r.GetContainerId())
	defer func() {
		if retErr == nil {
			glog.V(2).Infof("StartContainer %q returns successfully", r.GetContainerId())
		}
	}()

	container, err := c.containerStore.Get(r.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("an error occurred when try to find container %q: %v", r.GetContainerId(), err)
	}
	id := container.ID

	var startErr error
	// update container status in one transaction to avoid race with event monitor.
	if err := container.Status.Update(func(status containerstore.Status) (containerstore.Status, error) {
		// Always apply status change no matter startContainer fails or not. Because startContainer
		// may change container state no matter it fails or succeeds.
		startErr = c.startContainer(ctx, container.Container, container.Metadata, &status)
		return status, nil
	}); startErr != nil {
		return nil, startErr
	} else if err != nil {
		return nil, fmt.Errorf("failed to update container %q metadata: %v", id, err)
	}
	return &runtime.StartContainerResponse{}, nil
}

// startContainer actually starts the container. The function needs to be run in one transaction. Any updates
// to the status passed in will be applied no matter the function returns error or not.
func (c *criContainerdService) startContainer(ctx context.Context,
	container containerd.Container,
	meta containerstore.Metadata,
	status *containerstore.Status) (retErr error) {
	config := meta.Config
	id := container.ID()

	// Return error if container is not in created state.
	if status.State() != runtime.ContainerState_CONTAINER_CREATED {
		return fmt.Errorf("container %q is in %s state", id, criContainerStateToString(status.State()))
	}
	// Do not start the container when there is a removal in progress.
	if status.Removing {
		return fmt.Errorf("container %q is in removing state", id)
	}

	defer func() {
		if retErr != nil {
			// Set container to exited if fail to start.
			status.Pid = 0
			status.FinishedAt = time.Now().UnixNano()
			status.ExitCode = errorStartExitCode
			status.Reason = errorStartReason
			status.Message = retErr.Error()
		}
	}()

	// Get sandbox config from sandbox store.
	sandbox, err := c.sandboxStore.Get(meta.SandboxID)
	if err != nil {
		return fmt.Errorf("sandbox %q not found: %v", meta.SandboxID, err)
	}
	sandboxConfig := sandbox.Config
	sandboxID := meta.SandboxID
	// Make sure sandbox is running.
	s, err := sandbox.Container.Task(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get sandbox container %q info: %v", sandboxID, err)
	}
	// This is only a best effort check, sandbox may still exit after this. If sandbox fails
	// before starting the container, the start will fail.
	taskStatus, err := s.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get task status for sandbox container %q: %v", id, err)
	}

	if taskStatus.Status != containerd.Running {
		return fmt.Errorf("sandbox container %q is not running", sandboxID)
	}

	// Redirect the stream to std for now.
	// TODO(random-liu): [P1] Support StdinOnce after container logging is added.
	rStdoutPipe, wStdoutPipe := io.Pipe()
	rStderrPipe, wStderrPipe := io.Pipe()
	stdin := new(bytes.Buffer)
	defer func() {
		if retErr != nil {
			rStdoutPipe.Close()
			rStderrPipe.Close()
		}
	}()
	if config.GetLogPath() != "" {
		// Only generate container log when log path is specified.
		logPath := filepath.Join(sandboxConfig.GetLogDirectory(), config.GetLogPath())
		if err = c.agentFactory.NewContainerLogger(logPath, agents.Stdout, rStdoutPipe).Start(); err != nil {
			return fmt.Errorf("failed to start container stdout logger: %v", err)
		}
		// Only redirect stderr when there is no tty.
		if !config.GetTty() {
			if err = c.agentFactory.NewContainerLogger(logPath, agents.Stderr, rStderrPipe).Start(); err != nil {
				return fmt.Errorf("failed to start container stderr logger: %v", err)
			}
		}
	}
	//TODO(Abhi): close stdin/pass a managed IOCreation
	task, err := container.NewTask(ctx, containerd.NewIO(stdin, wStdoutPipe, wStderrPipe))
	if err != nil {
		return fmt.Errorf("failed to create containerd task: %v", err)
	}
	defer func() {
		if retErr != nil {
			if _, err := task.Delete(ctx, containerd.WithProcessKill); err != nil {
				glog.Errorf("Failed to delete containerd task %q: %v", id, err)
			}
		}
	}()

	// Start containerd task.
	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start containerd task %q: %v", id, err)
	}

	// Update container start timestamp.
	status.Pid = task.Pid()
	status.StartedAt = time.Now().UnixNano()
	return nil
}
