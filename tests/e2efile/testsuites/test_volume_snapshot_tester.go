/**
 * Copyright 2022 IBM Corp.
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
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
)

// DynamicallyProvisionedVolumeSnapshotTest will provision required StorageClass(es),VolumeSnapshotClass(es), PVC(s) and Pod(s)
// Waiting for the PV provisioner to create a new PV
// Testing if the Pod(s) can write and read to mounted volumes
// Create a snapshot, validate the data is still on the disk, and then write and read to it again
// And finally delete the snapshot
// This test only supports a single volume

type DynamicallyProvisionedVolumeSnapshotTest struct {
	Pod         PodDetails
	RestoredPod PodDetails
	PodCheck    *PodExecCheck
	PVCFail     bool
}

func (t *DynamicallyProvisionedVolumeSnapshotTest) Run(client clientset.Interface, restclient restclientset.Interface, namespace *v1.Namespace) {
	By("Executing Positive test scenario for volume snapshot")
	tpod := NewTestPod(client, namespace, t.Pod.Cmd)
	volume := t.Pod.Volumes[0]
	tpvc, pvcCleanup := volume.SetupDynamicPersistentVolumeClaim(client, namespace, false)
	for i := range pvcCleanup {
		defer pvcCleanup[i]()
	}
	tpod.SetupVolume(tpvc.persistentVolumeClaim, volume.VolumeMount.NameGenerate+"1", volume.VolumeMount.MountPathGenerate+"1", volume.VolumeMount.ReadOnly)

	By("deploying the pod")
	tpod.Create()
	defer tpod.Cleanup()
	By("checking that the pods command exits with no error")
	tpod.WaitForSuccess()

	By("taking snapshots")
	tvsc, cleanup := CreateVolumeSnapshotClass(restclient, namespace)
	defer cleanup()

	snapshot := tvsc.CreateSnapshot(tpvc.persistentVolumeClaim)
	defer tvsc.DeleteSnapshot(snapshot)
	tvsc.ReadyToUse(snapshot, false)
	By("Snapshot Creation Completed")
	t.RestoredPod.Volumes[0].DataSource = &DataSource{Name: snapshot.Name}
	trpod := NewTestPod(client, namespace, t.RestoredPod.Cmd)
	rvolume := t.RestoredPod.Volumes[0]
	By("Creating PersistentVolumeClaim from a Volume Snapshot")
	trpvc, rpvcCleanup := rvolume.SetupDynamicPersistentVolumeClaim(client, namespace, false)
	for i := range rpvcCleanup {
		defer rpvcCleanup[i]()
	}
	trpod.SetupVolume(trpvc.persistentVolumeClaim, rvolume.VolumeMount.NameGenerate+"1", rvolume.VolumeMount.MountPathGenerate+"1", rvolume.VolumeMount.ReadOnly)

	By("deploying a second pod with a volume restored from the snapshot")
	trpod.Create()
	defer trpod.Cleanup()
	By("checking that the pods command exits with no error")
	trpod.WaitForRunningSlow()
	trpod.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString01)
}

func (t *DynamicallyProvisionedVolumeSnapshotTest) VolumeSizeLess(client clientset.Interface, restclient restclientset.Interface, namespace *v1.Namespace) {
	By("Executing Negative test scenario for volume snapshot")
	tpod := NewTestPod(client, namespace, t.Pod.Cmd)
	volume := t.Pod.Volumes[0]
	tpvc, pvcCleanup := volume.SetupDynamicPersistentVolumeClaim(client, namespace, false)
	tpod.SetupVolume(tpvc.persistentVolumeClaim, volume.VolumeMount.NameGenerate+"1", volume.VolumeMount.MountPathGenerate+"1", volume.VolumeMount.ReadOnly)

	By("deploying the pod")
	tpod.Create()
	By("checking that the pods command exits with no error")
	tpod.WaitForSuccess()

	By("taking snapshots")
	tvsc, cleanup := CreateVolumeSnapshotClass(restclient, namespace)
	defer cleanup()

	snapshot := tvsc.CreateSnapshot(tpvc.persistentVolumeClaim)
	//defer tvsc.DeleteSnapshot(snapshot)
	tvsc.ReadyToUse(snapshot, false)
	By("Snapshot Creation Completed")
	t.RestoredPod.Volumes[0].DataSource = &DataSource{Name: snapshot.Name}
	rvolume := t.RestoredPod.Volumes[0]
	By("Creating PersistentVolumeClaim from a Volume Snapshot")
	_, rpvcCleanup := rvolume.SetupDynamicPersistentVolumeClaim(client, namespace, true)
	// Delete snapshot for which volume is deleted | detached
	defer tvsc.DeleteSnapshot(snapshot)
	for i := range pvcCleanup {
		defer pvcCleanup[i]()
	}
	defer tpod.Cleanup()
	for i := range rpvcCleanup {
		defer rpvcCleanup[i]()
	}
}

func (t *DynamicallyProvisionedVolumeSnapshotTest) SnapShotForUnattached(client clientset.Interface, restclient restclientset.Interface, namespace *v1.Namespace) {
	By("Executing Snapshot for Unattached volume")
	volume := t.Pod.Volumes[0]
	tpvc, pvcCleanup := volume.SetupDynamicPersistentVolumeClaim(client, namespace, false)
	for i := range pvcCleanup {
		defer pvcCleanup[i]()
	}
	By("taking snapshots")
	tvsc, cleanup := CreateVolumeSnapshotClass(restclient, namespace)
	defer cleanup()
	snapshot := tvsc.CreateSnapshot(tpvc.persistentVolumeClaim)
	//defer time.Sleep(13 * time.Minute)
	defer tvsc.DeleteSnapshot(snapshot)
	tvsc.ReadyToUse(snapshot, true)
}
