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

package e2e

import (
	"context"
	"os"

	"github.com/IBM/ibm-csi-common/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("[ics-e2e] [xfs] [sc] Dynamic Provisioning for XFS Filesystem", func() {
	var (
		cs       clientset.Interface
		fw       *framework.Framework
		ns       *v1.Namespace
		fpointer *os.File
		err      error
	)

	testResultFile := os.Getenv("E2E_TEST_RESULT")
	fw = framework.NewDefaultFramework("ics-e2e-xfs")

	BeforeEach(func() {
		cs = fw.ClientSet
		ns = fw.Namespace
	})

	// XFS General Purpose - POD Test
	It("[xfs-gp-pod] with XFS GP profile: should create pvc & pv, pod resources, write and read", func() {
		CreateXFSStorageClass("xfs-gp-test-sc", "general-purpose", "", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-gp-test-sc", metav1.DeleteOptions{})

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pods := []testsuites.PodDetails{
			{
				Cmd:      "df -h; mount | grep xfs; echo 'xfs gp test' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "xfs-gp-test-pvc",
						VolumeType:    "xfs-gp-test-sc",
						FSType:        "xfs",
						ClaimSize:     "11Gi",
						ReclaimPolicy: &reclaimPolicy,
						MountOptions:  []string{"rw"},
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}

		test := testsuites.DynamicallyProvisionePodWithVolTest{
			Pods: pods,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "xfs gp test\n",
				ExpectedString02: "xfs gp test\nxfs gp test\n",
			},
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS GP Profile POD Test: PASS\n"); err != nil {
			panic(err)
		}
	})

	// XFS General Purpose - DEPLOYMENT Test
	It("[xfs-gp-deploy] with XFS GP profile: deployment resources, write and read", func() {
		CreateXFSStorageClass("xfs-gp-deploy-sc", "general-purpose", "", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-gp-deploy-sc", metav1.DeleteOptions{})

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pod := testsuites.PodDetails{
			Cmd:      "echo 'xfs gp deployment' >> /mnt/test-1/data && while true; do sleep 2; done",
			CmdExits: false,
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "xfs-gp-deploy-pvc",
					VolumeType:    "xfs-gp-deploy-sc",
					FSType:        "xfs",
					ClaimSize:     "11Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}

		test := testsuites.DynamicallyProvisioneDeployWithVolWRTest{
			Pod: pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "xfs gp deployment\n",
				ExpectedString02: "xfs gp deployment\nxfs gp deployment\n",
			},
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS GP Profile DEPLOYMENT Test: PASS\n"); err != nil {
			panic(err)
		}
	})

	// XFS 5iops - POD Test
	It("[xfs-5iops-pod] with XFS 5iops profile: should create pvc & pv, pod resources", func() {
		CreateXFSStorageClass("xfs-5iops-test-sc", "5iops-tier", "", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-5iops-test-sc", metav1.DeleteOptions{})

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pods := []testsuites.PodDetails{
			{
				Cmd:      "xfs_info /mnt/test-1; echo 'xfs 5iops test' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "xfs-5iops-test-pvc",
						VolumeType:    "xfs-5iops-test-sc",
						FSType:        "xfs",
						ClaimSize:     "10Gi",
						ReclaimPolicy: &reclaimPolicy,
						MountOptions:  []string{"rw"},
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}

		test := testsuites.DynamicallyProvisionePodWithVolTest{
			Pods: pods,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "xfs 5iops test\n",
				ExpectedString02: "xfs 5iops test\nxfs 5iops test\n",
			},
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS 5iops Profile POD Test: PASS\n"); err != nil {
			panic(err)
		}
	})

	// XFS 5iops - DEPLOYMENT Test
	It("[xfs-5iops-deploy] with XFS 5iops profile: deployment resources", func() {
		CreateXFSStorageClass("xfs-5iops-deploy-sc", "5iops-tier", "", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-5iops-deploy-sc", metav1.DeleteOptions{})

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pod := testsuites.PodDetails{
			Cmd:      "echo 'xfs 5iops deployment' >> /mnt/test-1/data && while true; do sleep 2; done",
			CmdExits: false,
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "xfs-5iops-deploy-pvc",
					VolumeType:    "xfs-5iops-deploy-sc",
					FSType:        "xfs",
					ClaimSize:     "10Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}

		test := testsuites.DynamicallyProvisioneDeployWithVolWRTest{
			Pod: pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "xfs 5iops deployment\n",
				ExpectedString02: "xfs 5iops deployment\nxfs 5iops deployment\n",
			},
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS 5iops Profile DEPLOYMENT Test: PASS\n"); err != nil {
			panic(err)
		}
	})

	// XFS General Purpose - POD Resize Test
	It("[xfs-gp-resize] with XFS GP profile: should resize volume", func() {
		CreateXFSStorageClass("xfs-gp-resize-sc", "general-purpose", "", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-gp-resize-sc", metav1.DeleteOptions{})

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'xfs resize test' >> /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "xfs-gp-resize-pvc",
						VolumeType:    "xfs-gp-resize-sc",
						FSType:        "xfs",
						ClaimSize:     "11Gi",
						ReclaimPolicy: &reclaimPolicy,
						MountOptions:  []string{"rw"},
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}

		test := testsuites.DynamicallyProvisionedResizeVolumeTest{
			Pods: pods,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "xfs resize test\n",
				ExpectedString02: "xfs resize test\nxfs resize test\n",
			},
			ExpandVolSizeG: 20,
			ExpandedSize:   19,
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS GP Profile POD Resize Volume: PASS\n"); err != nil {
			panic(err)
		}
	})

	// XFS 5iops - POD Resize Test
	It("[xfs-5iops-resize] with XFS 5iops profile: should resize volume", func() {
		CreateXFSStorageClass("xfs-5iops-resize-sc", "5iops-tier", "", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-5iops-resize-sc", metav1.DeleteOptions{})

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'xfs 5iops resize test' >> /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "xfs-5iops-resize-pvc",
						VolumeType:    "xfs-5iops-resize-sc",
						FSType:        "xfs",
						ClaimSize:     "10Gi",
						ReclaimPolicy: &reclaimPolicy,
						MountOptions:  []string{"rw"},
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}

		test := testsuites.DynamicallyProvisionedResizeVolumeTest{
			Pods: pods,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "xfs 5iops resize test\n",
				ExpectedString02: "xfs 5iops resize test\nxfs 5iops resize test\n",
			},
			ExpandVolSizeG: 15,
			ExpandedSize:   14,
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS 5iops Profile POD Resize Volume: PASS\n"); err != nil {
			panic(err)
		}
	})
})
