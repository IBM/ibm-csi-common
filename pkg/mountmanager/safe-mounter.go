/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package mountmanager ...
package mountmanager

import (
	mount "k8s.io/mount-utils"
	exec "k8s.io/utils/exec"
)

type mountInterface = mount.Interface

// Mounter is the interface implemented by Mounter
type Mounter interface {
	mountInterface

	MountEITBasedFileShare(mountPath string, targetPath string, fsType string, requestID string) (string, error)
	GetSafeFormatAndMount() *mount.SafeFormatAndMount
	MakeFile(path string) error
	MakeDir(path string) error
	PathExists(path string) (bool, error)
	Resize(string, string) (bool, error)
}

// NodeMounter implements Mounter.
// A superstruct of SafeFormatAndMount.
type NodeMounter struct {
	*mount.SafeFormatAndMount
}

// NewNodeMounter ...
func NewNodeMounter() Mounter {
	// mounter.newSafeMounter returns a SafeFormatAndMount
	safeMounter := newSafeMounter()
	return &NodeMounter{safeMounter}
}

// NewSafeMounter ...
func newSafeMounter() *mount.SafeFormatAndMount {
	realMounter := mount.New("")
	realExec := exec.New()
	return &mount.SafeFormatAndMount{
		Interface: realMounter,
		Exec:      realExec,
	}
}
