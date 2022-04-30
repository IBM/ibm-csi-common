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

package testsuites

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	clientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
)

type PodDetails struct {
	Cmd      string
	CmdExits bool
	Volumes  []VolumeDetails
}

type VolumeMode int

type VolumeDetails struct {
	PVCName               string //PVC Name
	VolumeType            string //PVC SC
	FSType                string //Ext3 / XFS / EXT4
	Encrypted             bool
	MountOptions          []string
	ClaimSize             string //PVC Capacity
	ReclaimPolicy         *v1.PersistentVolumeReclaimPolicy
	VolumeBindingMode     *storagev1.VolumeBindingMode
	AllowedTopologyValues []string
	AccessMode            *v1.PersistentVolumeAccessMode
	VolumeMode            VolumeMode
	VolumeMount           VolumeMountDetails
	VolumeDevice          VolumeDeviceDetails
	pvc                   *TestPersistentVolumeClaim
	DataSource            *DataSource
}

const (
	FileSystem VolumeMode = iota
	Block
)

const (
	VolumeSnapshotKind      = "VolumeSnapshot"
	SnapshotAPIVersion      = "snapshot.storage.k8s.io/v1"
	APIVersionv1            = "v1"
	VolumeSnapshotClassKind = "VolumeSnapshotClass"
)

var (
	SnapshotAPIGroup = "snapshot.storage.k8s.io"
)

type VolumeMountDetails struct {
	NameGenerate      string
	MountPathGenerate string
	ReadOnly          bool
}

type VolumeDeviceDetails struct {
	NameGenerate string
	DevicePath   string
}

type DataSource struct {
	Name string
}

type PodExecCheck struct {
	Cmd              []string
	ExpectedString01 string
	ExpectedString02 string
}

func (pod *PodDetails) SetupWithPVC(client clientset.Interface, namespace *v1.Namespace, name string) (*TestPod, []func()) {
	tpod := NewTestPodWithName(client, namespace, name, pod.Cmd)
	cleanupFuncs := make([]func(), 0)
	for n, v := range pod.Volumes {
		if v.pvc == nil {
			continue
		}
		tpod.SetupVolume(v.pvc.persistentVolumeClaim,
			fmt.Sprintf("%s%d", v.VolumeMount.NameGenerate, n+1),
			fmt.Sprintf("%s%d", v.VolumeMount.MountPathGenerate, n+1), v.VolumeMount.ReadOnly)
	}
	return tpod, cleanupFuncs
}

func (pod *PodDetails) SetupWithDynamicVolumes(client clientset.Interface, namespace *v1.Namespace) (*TestPod, []func()) {
	cleanupFuncs := make([]func(), 0)

	By("setting up POD")
	tpod := NewTestPod(client, namespace, pod.Cmd)
	By("setting up the PVC for POD")
	for n, v := range pod.Volumes {
		tpvc, funcs := v.SetupDynamicPersistentVolumeClaim(client, namespace, false)
		cleanupFuncs = append(cleanupFuncs, funcs...)

		if v.VolumeMode == Block {
			tpod.SetupRawBlockVolume(tpvc.persistentVolumeClaim,
				fmt.Sprintf("%s%d", v.VolumeDevice.NameGenerate, n+1), v.VolumeDevice.DevicePath)
		} else {
			tpod.SetupVolume(tpvc.persistentVolumeClaim,
				fmt.Sprintf("%s%d", v.VolumeMount.NameGenerate, n+1),
				fmt.Sprintf("%s%d", v.VolumeMount.MountPathGenerate, n+1), v.VolumeMount.ReadOnly)
		}
	}
	return tpod, cleanupFuncs
}

func (pod *PodDetails) SetupDeploymentWithMultiVol(client clientset.Interface, namespace *v1.Namespace) (*TestDeployment, []func()) {
	cleanupFuncs := make([]func(), 0)
	By("setting up the Deployment")
	tDeployment := NewTestDeploymentWitoutPVC(client, namespace, pod.Cmd)

	for n, volume := range pod.Volumes {
		By("setting up the PVC for Deployment")
		storageClass := storagev1.StorageClass{}
		storageClass.Name = volume.VolumeType
		storageClass.ReclaimPolicy = volume.ReclaimPolicy
		storageClass.MountOptions = volume.MountOptions

		tpvc := NewTestPersistentVolumeClaim(client, volume.PVCName, namespace, volume.ClaimSize, volume.AccessMode, volume.VolumeMode, &storageClass)
		tpvc.Create()
		tpvc.WaitForBound()
		tpvc.ValidateProvisionedPersistentVolume()
		cleanupFuncs = append(cleanupFuncs, tpvc.Cleanup)
		tDeployment.SetupVolume(tpvc.persistentVolumeClaim,
			fmt.Sprintf("%s%d", volume.VolumeMount.NameGenerate, n+1),
			fmt.Sprintf("%s%d", volume.VolumeMount.MountPathGenerate, n+1), volume.VolumeMount.ReadOnly)
	}
	cleanupFuncs = append(cleanupFuncs, tDeployment.Cleanup)
	return tDeployment, cleanupFuncs
}

func (pod *PodDetails) SetupDeployment(client clientset.Interface, namespace *v1.Namespace) (*TestDeployment, []func()) {
	cleanupFuncs := make([]func(), 0)
	volume := pod.Volumes[0]

	By("setting up the PVC for Deployment")
	storageClass := storagev1.StorageClass{}
	storageClass.Name = volume.VolumeType
	storageClass.ReclaimPolicy = volume.ReclaimPolicy
	storageClass.MountOptions = volume.MountOptions

	tpvc := NewTestPersistentVolumeClaim(client, volume.PVCName, namespace, volume.ClaimSize, volume.AccessMode, volume.VolumeMode, &storageClass)
	tpvc.Create()
	tpvc.WaitForBound()
	tpvc.ValidateProvisionedPersistentVolume()
	cleanupFuncs = append(cleanupFuncs, tpvc.Cleanup)

	By("setting up the Deployment")
	tDeployment := NewTestDeployment(client, namespace, pod.Cmd,
		tpvc.persistentVolumeClaim,
		fmt.Sprintf("%s%d", volume.VolumeMount.NameGenerate, 1),
		fmt.Sprintf("%s%d", volume.VolumeMount.MountPathGenerate, 1),
		volume.VolumeMount.ReadOnly)

	cleanupFuncs = append(cleanupFuncs, tDeployment.Cleanup)
	return tDeployment, cleanupFuncs
}

func (volume *VolumeDetails) SetupDynamicPersistentVolumeClaim(client clientset.Interface, namespace *v1.Namespace, pvcErrExpected bool) (*TestPersistentVolumeClaim, []func()) {
	cleanupFuncs := make([]func(), 0)
	By("setting up the PVC and PV")
	//By(fmt.Sprintf("PVC: %q    NS: %q", volume.PVCName, namespace.Name))
	storageClass := storagev1.StorageClass{}
	storageClass.Name = volume.VolumeType
	storageClass.ReclaimPolicy = volume.ReclaimPolicy
	storageClass.MountOptions = volume.MountOptions

	var tpvc *TestPersistentVolumeClaim
	if volume.DataSource != nil {
		By("Setting up datasource in PVC")
		dataSource := &v1.TypedLocalObjectReference{
			Name:     volume.DataSource.Name,
			Kind:     VolumeSnapshotKind,
			APIGroup: &SnapshotAPIGroup,
		}
		tpvc = NewTestPersistentVolumeClaimWithDataSource(client, volume.PVCName, namespace, volume.ClaimSize, volume.VolumeMode, &storageClass, dataSource)
		By(fmt.Sprintf("%q", tpvc))
	} else {
		tpvc = NewTestPersistentVolumeClaim(client, volume.PVCName, namespace, volume.ClaimSize, volume.AccessMode, volume.VolumeMode, &storageClass)
	}
	tpvc.Create()
	cleanupFuncs = append(cleanupFuncs, tpvc.Cleanup)
	// PV will not be ready until PVC is used in a pod when volumeBindingMode: WaitForFirstConsumer
	if volume.VolumeBindingMode == nil || *volume.VolumeBindingMode == storagev1.VolumeBindingImmediate {
		if pvcErrExpected == true {
			By("PVC Creation should go to Pending state as volume size is less than source volume")
			tpvc.WaitForPending()
		} else {
			tpvc.WaitForBound()
			tpvc.ValidateProvisionedPersistentVolume()
		}
	}
	volume.pvc = tpvc

	return tpvc, cleanupFuncs
}
func (pod *PodDetails) SetupStatefulset(client clientset.Interface, namespace *v1.Namespace, serviceName string, labels map[string]string, replicaCount int32) (*TestStatefulsets, []func()) {
	cleanupFuncs := make([]func(), 0)
	volume := pod.Volumes[0]
	storageClass := storagev1.StorageClass{}
	storageClass.Name = volume.VolumeType
	storageClass.ReclaimPolicy = volume.ReclaimPolicy
	storageClass.MountOptions = volume.MountOptions
	By("Setting up PVC values")
	tpvc := NewTestPersistentVolumeClaim(client, volume.PVCName, namespace, volume.ClaimSize, volume.AccessMode, volume.VolumeMode, &storageClass)
	By("Setting up the Statefulset")
	tStatefulset := tpvc.NewTestStatefulset(client, namespace, serviceName, pod.Cmd,
		storageClass.Name,
		volume.PVCName,
		fmt.Sprintf("%s%d", volume.VolumeMount.MountPathGenerate, 1), labels, replicaCount)

	cleanupFuncs = append(cleanupFuncs, tStatefulset.Cleanup)
	return tStatefulset, cleanupFuncs
}

func CreateVolumeSnapshotClass(client restclientset.Interface, namespace *v1.Namespace) (*TestVolumeSnapshotClass, func()) {
	By("setting up the VolumeSnapshotClass")
	volumeSnapshotClass := GetVolumeSnapshotClass(namespace.Name)
	tvsc := NewTestVolumeSnapshotClass(client, namespace, volumeSnapshotClass)
	tvsc.Create()

	return tvsc, tvsc.Cleanup
}
