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
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri/v1alpha1/runtime"

	content "github.com/containerd/containerd/content"
	imagestore "github.com/kubernetes-incubator/cri-containerd/pkg/store/image"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageStatus returns the status of the image, returns nil if the image isn't present.
// TODO(random-liu): We should change CRI to distinguish image id and image spec. (See
// kubernetes/kubernetes#46255)
func (c *criContainerdService) ImageStatus(ctx context.Context, r *runtime.ImageStatusRequest) (*runtime.ImageStatusResponse, error) {
	image, err := c.localResolve(ctx, r.GetImage().GetImage())
	if err != nil {
		return nil, fmt.Errorf("can not resolve %q locally: %v", r.GetImage().GetImage(), err)
	}
	if image == nil {
		// return empty without error when image not found.
		return &runtime.ImageStatusResponse{}, nil
	}
	// TODO(random-liu): [P0] Make sure corresponding snapshot exists. What if snapshot
	// doesn't exist?

	runtimeImage := toCRIRuntimeImage(image)
	info, err := c.toCRIImageInfo(ctx, image, r.GetVerbose())
	if err != nil {
		return nil, fmt.Errorf("failed to generate image info: %v", err)
	}

	return &runtime.ImageStatusResponse{
		Image: runtimeImage,
		Info:  info,
	}, nil
}

// toCRIRuntimeImage converts internal image object to CRI runtime.Image.
func toCRIRuntimeImage(image *imagestore.Image) *runtime.Image {
	runtimeImage := &runtime.Image{
		Id:          image.ID,
		RepoTags:    image.RepoTags,
		RepoDigests: image.RepoDigests,
		Size_:       uint64(image.Size),
	}
	uid, username := getUserFromImage(image.Config.User)
	if uid != nil {
		runtimeImage.Uid = &runtime.Int64Value{Value: *uid}
	}
	runtimeImage.Username = username

	return runtimeImage
}

// TODO (mikebrow): discuss moving this struct and / or constants for info map for some or all of these fields to CRI
type verboseImageInfo struct {
	Config             *imagespec.ImageConfig `json:"config"`
	ConfigDescriptor   imagespec.Descriptor   `json:"configDescriptor"`
	ManifestDescriptor imagespec.Descriptor   `json:"manifestDescriptor"`
	LayerInfo          []content.Info         `json:"layerInfo"`
}

// toCRIImageInfo converts internal image object information to CRI image status response info map.
func (c *criContainerdService) toCRIImageInfo(ctx context.Context, image *imagestore.Image, verbose bool) (map[string]string, error) {
	if !verbose {
		return nil, nil
	}

	info := make(map[string]string)
	i := image.Image
	descriptor, err := i.Config(ctx)
	if err != nil {
		glog.Errorf("Failed to get image config %q: %v", image.ID, err)
	} // fallthrough

	targetDescriptor := i.Target()
	var dia []content.Info
	digests, err := i.RootFS(ctx)
	if err != nil {
		glog.Errorf("Failed to get target digests %q: %v", i.Name(), err)
	} else {
		dia = make([]content.Info, len(digests))
		for i, d := range digests {
			di, err := c.client.ContentStore().Info(ctx, d)
			if err == nil {
				dia[i] = di
			}
		}
	}

	imi := &verboseImageInfo{
		Config:             image.Config,
		ConfigDescriptor:   descriptor,
		ManifestDescriptor: targetDescriptor,
		LayerInfo:          dia,
	}

	m, err := json.Marshal(imi)
	if err == nil {
		info["info"] = string(m)
	} else {
		glog.Errorf("failed to marshal info %v: %v", imi, err)
		info["info"] = err.Error()
	}

	return info, nil
}
