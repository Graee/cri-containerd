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

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/docker/docker/pkg/system"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri/v1alpha1/runtime"

	"github.com/kubernetes-incubator/cri-containerd/pkg/store"
	containerstore "github.com/kubernetes-incubator/cri-containerd/pkg/store/container"
)

// RemoveContainer removes the container.
// TODO(random-liu): Forcibly stop container if it's running.
func (c *criContainerdService) RemoveContainer(ctx context.Context, r *runtime.RemoveContainerRequest) (_ *runtime.RemoveContainerResponse, retErr error) {
	container, err := c.containerStore.Get(r.GetContainerId())
	if err != nil {
		if err != store.ErrNotExist {
			return nil, fmt.Errorf("an error occurred when try to find container %q: %v", r.GetContainerId(), err)
		}
		// Do not return error if container metadata doesn't exist.
		glog.V(5).Infof("RemoveContainer called for container %q that does not exist", r.GetContainerId())
		return &runtime.RemoveContainerResponse{}, nil
	}
	id := container.ID

	// Set removing state to prevent other start/remove operations against this container
	// while it's being removed.
	if err := setContainerRemoving(container); err != nil {
		return nil, fmt.Errorf("failed to set removing state for container %q: %v", id, err)
	}
	defer func() {
		if retErr != nil {
			// Reset removing if remove failed.
			if err := resetContainerRemoving(container); err != nil {
				glog.Errorf("failed to reset removing state for container %q: %v", id, err)
			}
		}
	}()

	// NOTE(random-liu): Docker set container to "Dead" state when start removing the
	// container so as to avoid start/restart the container again. However, for current
	// kubelet implementation, we'll never start a container once we decide to remove it,
	// so we don't need the "Dead" state for now.

	// Delete containerd container.
	if err := container.Container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, fmt.Errorf("failed to delete containerd container %q: %v", id, err)
		}
		glog.V(5).Infof("Remove called for containerd container %q that does not exist", id, err)
	}

	// Delete container checkpoint.
	if err := container.Delete(); err != nil {
		return nil, fmt.Errorf("failed to delete container checkpoint for %q: %v", id, err)
	}

	containerRootDir := getContainerRootDir(c.config.RootDir, id)
	if err := system.EnsureRemoveAll(containerRootDir); err != nil {
		return nil, fmt.Errorf("failed to remove container root directory %q: %v",
			containerRootDir, err)
	}

	c.containerStore.Delete(id)

	c.containerNameIndex.ReleaseByKey(id)

	return &runtime.RemoveContainerResponse{}, nil
}

// setContainerRemoving sets the container into removing state. In removing state, the
// container will not be started or removed again.
func setContainerRemoving(container containerstore.Container) error {
	return container.Status.Update(func(status containerstore.Status) (containerstore.Status, error) {
		// Do not remove container if it's still running.
		if status.State() == runtime.ContainerState_CONTAINER_RUNNING {
			return status, fmt.Errorf("container is still running")
		}
		if status.Removing {
			return status, fmt.Errorf("container is already in removing state")
		}
		status.Removing = true
		return status, nil
	})
}

// resetContainerRemoving resets the container removing state on remove failure. So
// that we could remove the container again.
func resetContainerRemoving(container containerstore.Container) error {
	return container.Status.Update(func(status containerstore.Status) (containerstore.Status, error) {
		status.Removing = false
		return status, nil
	})
}
