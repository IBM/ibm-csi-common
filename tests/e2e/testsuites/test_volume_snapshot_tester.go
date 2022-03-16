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
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"

	. "github.com/onsi/ginkgo"
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
}

func (t *DynamicallyProvisionedVolumeSnapshotTest) Run(client clientset.Interface, restclient restclientset.Interface, namespace *v1.Namespace) {
	for n, pod := range t.Pods {
		tpod, cleanup := pod.SetupWithDynamicVolumes(client, namespace)
		// defer must be called here for resources not get removed before using them
		for i := range cleanup {
			defer cleanup[i]()
		}
		ind := fmt.Sprintf("%d", n)
		By(fmt.Sprintf("deploying the pod %q", ind))
		tpod.Create()
		defer tpod.Cleanup()

		if !pod.CmdExits {
			By("checking that the pods status is running")
			tpod.WaitForRunningSlow()
			if t.PodCheck != nil {
				By("checking pod exec after pod recreate")
				tpod.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString01)
			}
		} else {
			By("checking that the pods command exits with no error")
			tpod.WaitForSuccess()
		}

		By("taking snapshots")
		tvsc, cleanup := CreateVolumeSnapshotClass(restclient, namespace)
		defer cleanup()

		snapshot := tvsc.CreateSnapshot(tpvc.persistentVolumeClaim)
		defer tvsc.DeleteSnapshot(snapshot)
		tvsc.ReadyToUse(snapshot)

		t.RestoredPod.Volumes[0].DataSource = &DataSource{Name: snapshot.Name}
		trpod := NewTestPod(client, namespace, t.RestoredPod.Cmd)
		rvolume := t.RestoredPod.Volumes[0]
		trpvc, rpvcCleanup := rvolume.SetupDynamicPersistentVolumeClaim(client, namespace)
		for i := range rpvcCleanup {
			defer rpvcCleanup[i]()
		}
		trpod.SetupVolume(trpvc.persistentVolumeClaim, rvolume.VolumeMount.NameGenerate+"1", rvolume.VolumeMount.MountPathGenerate+"1", rvolume.VolumeMount.ReadOnly)

		By("deploying a second pod with a volume restored from the snapshot")
		trpod.Create()
		defer trpod.Cleanup()
		By("checking that the pods command exits with no error")
		trpod.WaitForSuccess()
	}
}
