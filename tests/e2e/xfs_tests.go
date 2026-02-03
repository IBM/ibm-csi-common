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

	// Test 1: XFS with Tier Profile - Pod
	It("[xfs-tier-pod] with XFS tier profile: should create pvc & pv, pod resources", func() {
		CreateStorageClass("xfs-5iops-deploy-sc", "5iops-tier", "xfs", "", "", cs)
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

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS Tier Profile Test: PASS\n"); err != nil {
			panic(err)
		}
	})

	// Test 2: XFS with Tier Profile - Resize
	It("[xfs-tier-resize] with XFS tier profile: should resize volume", func() {
		CreateStorageClass("xfs-5iops-deploy-sc", "5iops-tier", "xfs", "", "", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-5iops-deploy-sc", metav1.DeleteOptions{})

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

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS Tier Profile Resize: PASS\n"); err != nil {
			panic(err)
		}
	})

	// Test 3: XFS with SDP Profile - Pod
	It("[xfs-sdp-pod] with XFS SDP profile: should create pvc & pv, pod resources", func() {
		CreateStorageClass("xfs-sdp-test-sc", "sdp", "xfs", "3000", "1000", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-sdp-test-sc", metav1.DeleteOptions{})

		// Create secret for SDP
		secretKey := os.Getenv("IC_API_KEY_STAG")
		if secretKey == "" {
			Skip("Skipping SDP test - IC_API_KEY_STAG not set")
		}

		secret := testsuites.NewSecret(cs, "xfs-sdp-test-pvc", ns.Name, "800", "e2e test",
			"false", secretKey, "vpc.block.csi.ibm.io")
		secret.Create()
		defer secret.Cleanup()

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pods := []testsuites.PodDetails{
			{
				Cmd:      "xfs_info /mnt/test-1; echo 'xfs sdp test' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "xfs-sdp-test-pvc",
						VolumeType:    "xfs-sdp-test-sc",
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
				ExpectedString01: "xfs sdp test\n",
				ExpectedString02: "xfs sdp test\nxfs sdp test\n",
			},
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS SDP Profile Test: PASS\n"); err != nil {
			panic(err)
		}
	})

	// Test 4: XFS with SDP Profile - Resize
	It("[xfs-sdp-resize] with XFS SDP profile: should resize volume", func() {
		CreateStorageClass("xfs-sdp-resize-sc", "sdp", "xfs", "3000", "1000", cs)
		defer cs.StorageV1().StorageClasses().Delete(context.Background(), "xfs-sdp-resize-sc", metav1.DeleteOptions{})

		// Create secret for SDP
		secretKey := os.Getenv("IC_API_KEY_STAG")
		if secretKey == "" {
			Skip("Skipping SDP resize test - IC_API_KEY_STAG not set")
		}

		secret := testsuites.NewSecret(cs, "xfs-sdp-resize-pvc", ns.Name, "800", "e2e test",
			"false", secretKey, "vpc.block.csi.ibm.io")
		secret.Create()
		defer secret.Cleanup()

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'xfs sdp resize test' >> /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "xfs-sdp-resize-pvc",
						VolumeType:    "xfs-sdp-resize-sc",
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
				ExpectedString01: "xfs sdp resize test\n",
				ExpectedString02: "xfs sdp resize test\nxfs sdp resize test\n",
			},
			ExpandVolSizeG: 15,
			ExpandedSize:   14,
		}
		test.Run(cs, ns)

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: XFS SDP Profile Resize: PASS\n"); err != nil {
			panic(err)
		}
	})
})
