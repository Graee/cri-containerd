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
	"fmt"
	"io"

	"github.com/containerd/containerd"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri/v1alpha1/runtime"

	cio "github.com/kubernetes-incubator/cri-containerd/pkg/server/io"
)

// Attach prepares a streaming endpoint to attach to a running container, and returns the address.
func (c *criContainerdService) Attach(ctx context.Context, r *runtime.AttachRequest) (*runtime.AttachResponse, error) {
	cntr, err := c.containerStore.Get(r.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("failed to find container in store: %v", err)
	}
	state := cntr.Status.Get().State()
	if state != runtime.ContainerState_CONTAINER_RUNNING {
		return nil, fmt.Errorf("container is in %s state", criContainerStateToString(state))
	}
	return c.streamServer.GetAttach(r)
}

func (c *criContainerdService) attachContainer(ctx context.Context, id string, stdin io.Reader, stdout, stderr io.WriteCloser,
	tty bool, resize <-chan remotecommand.TerminalSize) error {
	// Get container from our container store.
	cntr, err := c.containerStore.Get(id)
	if err != nil {
		return fmt.Errorf("failed to find container %q in store: %v", id, err)
	}
	id = cntr.ID

	state := cntr.Status.Get().State()
	if state != runtime.ContainerState_CONTAINER_RUNNING {
		return fmt.Errorf("container is in %s state", criContainerStateToString(state))
	}

	task, err := cntr.Container.Task(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to load task: %v", err)
	}
	handleResizing(resize, func(size remotecommand.TerminalSize) {
		if err := task.Resize(ctx, uint32(size.Width), uint32(size.Height)); err != nil {
			glog.Errorf("Failed to resize task %q console: %v", id, err)
		}
	})

	opts := cio.AttachOptions{
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		Tty:       tty,
		StdinOnce: cntr.Config.StdinOnce,
		CloseStdin: func() error {
			return task.CloseIO(ctx, containerd.WithStdinCloser)
		},
	}
	// TODO(random-liu): Figure out whether we need to support historical output.
	if err := cntr.IO.Attach(opts); err != nil {
		return fmt.Errorf("failed to attach container: %v", err)
	}
	return nil
}
