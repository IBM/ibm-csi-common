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
	testExec "k8s.io/utils/exec/testing"
)

// FakeNodeMounter ...
type FakeNodeMounter struct {
	*mount.SafeFormatAndMount
}

// NewFakeNodeMounter ...
func NewFakeNodeMounter() Mounter {
	//Have to make changes here to pass the Mock functions
	fakesafemounter := NewFakeSafeMounter()
	return &FakeNodeMounter{fakesafemounter}
}

// NewFakeSafeMounter ...
<<<<<<< HEAD
func NewFakeSafeMounter() *mount.SafeFormatAndMount {
=======
func NewFakeSafeMounter() Mounter {
	// execCallback := func(cmd string, args ...string) ([]byte, error) {
	// 	return nil, nil
	// }
>>>>>>> 065876c (Fix fake mounter)
	fakeMounter := &mount.FakeMounter{MountPoints: []mount.MountPoint{{
		Device: "valid-devicePath",
		Path:   "valid-vol-path",
		Type:   "ext4",
		Opts:   []string{"defaults"},
		Freq:   1,
		Pass:   2,
	}},
	}
<<<<<<< HEAD

	var fakeExec exec.Interface = &testExec.FakeExec{
		DisableScripts: true,
	}

	return &mount.SafeFormatAndMount{
=======
	//TODO: NewFakeExec is deprecated; failing test
	//fakeExec := exec.New()
	var fakeExecInterface exec.Interface = &testExec.FakeExec{}
	//mount.NewFakeExec(execCallback)
	return &NodeMounter{&mount.SafeFormatAndMount{
>>>>>>> 065876c (Fix fake mounter)
		Interface: fakeMounter,
		Exec:      fakeExecInterface,
	}}
}

// MakeDir ...
func (f *FakeNodeMounter) MakeDir(pathname string) error {
	return nil
}

// MakeFile ...
func (f *FakeNodeMounter) MakeFile(pathname string) error {
	return nil
}

// PathExists ...
func (f *FakeNodeMounter) PathExists(pathname string) (bool, error) {
	return mount.PathExists(pathname)
}

// NewSafeFormatAndMount ...
func (f *FakeNodeMounter) NewSafeFormatAndMount() *mount.SafeFormatAndMount {
	return NewFakeSafeMounter()
}
