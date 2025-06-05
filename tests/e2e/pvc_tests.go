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
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/IBM/ibm-csi-common/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo/v2"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"

	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
)

const defaultSecret = ""

var testResultFile = os.Getenv("E2E_TEST_RESULT")
var err error
var fpointer *os.File

var _ = Describe("[ics-e2e] [sc] [with-deploy] Dynamic Provisioning for all SC with Deployment", func() {
	f := framework.NewDefaultFramework("ics-e2e-deploy")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs        clientset.Interface
		ns        *v1.Namespace
		secretKey string
	)

	secretKey = os.Getenv("E2E_SECRET_ENCRYPTION_KEY")
	if secretKey == "" {
		secretKey = defaultSecret
	}

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
	})

	It("with 5iops sc: should create a pvc &pv, deployment resources, write and read to volume, delete the pod, write and read to volume again", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		pod := testsuites.PodDetails{
			Cmd:      "echo 'hello world' >> /mnt/test-1/data && while true; do sleep 2; done",
			CmdExits: false,
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-5iops-",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "15Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5IOPS SC DEPLOYMENT TEST: PASS\n"); err != nil {
			panic(err)
		}
	})

	It("with generalpurpose sc: should create a pvc & pv, deployment resources, write and read to volume, delete the pod, write and read to volume again", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pod := testsuites.PodDetails{
			Cmd:      "echo 'hello world' >> /mnt/test-1/data && while true; do sleep 2; done",
			CmdExits: false,
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-gp-",
					VolumeType:    "ibmc-vpc-block-general-purpose",
					FSType:        "ext4",
					ClaimSize:     "35Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: GP SC DEPLOYMENT TEST: PASS\n"); err != nil {
			panic(err)
		}
	})
})

var _ = Describe("[ics-e2e] [sc] [with-pods] Dynamic Provisioning for all SC with PODs", func() {
	f := framework.NewDefaultFramework("ics-e2e-pods")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs        clientset.Interface
		ns        *v1.Namespace
		secretKey string
	)

	secretKey = os.Getenv("E2E_SECRET_ENCRYPTION_KEY")
	if secretKey == "" {
		secretKey = defaultSecret
	}

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
	})

	It("with 5iops sc: should create a pvc & pv, pod resources, write and read to volume", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pods := []testsuites.PodDetails{
			{
				// Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Cmd:      "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "ics-vol-5iops-",
						VolumeType:    "ibmc-vpc-block-5iops-tier",
						FSType:        "ext4",
						ClaimSize:     "15Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5IOPS SC POD TEST: PASS\n"); err != nil {
			panic(err)
		}
	})

	It("with generalpurpose sc: should create a pvc & pv, pod resources, write and read to volume", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "ics-vol-gp-",
						VolumeType:    "ibmc-vpc-block-general-purpose",
						FSType:        "ext4",
						ClaimSize:     "35Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: GP SC POD TEST: PASS\n"); err != nil {
			panic(err)
		}
	})

	It("with custom sc: should create a pvc & pv, pod resources, write and read to volume", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		// For Custom SC PVC name and Secret name should be same and in same NS
		secret := testsuites.NewSecret(cs, "ics-vol-block-custom", ns.Name, "800", "e2e test",
			"false", secretKey, "vpc.block.csi.ibm.io")
		secret.Create()
		defer secret.Cleanup()
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "ics-vol-block-custom",
						VolumeType:    "ibmc-vpc-block-custom",
						FSType:        "ext4",
						ClaimSize:     "45Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: CUSTOM SC POD TEST: PASS\n"); err != nil {
			panic(err)
		}
	})
})

var _ = Describe("[ics-e2e] [sc] [with-statefulset] Dynamic Provisioning using statefulsets", func() {
	f := framework.NewDefaultFramework("ics-e2e-statefulsets")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs clientset.Interface
		ns *v1.Namespace
	)
	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
	})
	It("With 5iops sc: should creat statefuleset resources, write and read to volume", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		var replicaCount = int32(2)
		headlessService := testsuites.NewHeadlessService(cs, "ics-e2e-service-", ns.Name, "test")
		service := headlessService.Create()
		defer headlessService.Cleanup()

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		pod := testsuites.PodDetails{
			Cmd: "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-5iops",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "20Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}

		test := testsuites.StatefulsetWithVolWRTest{
			Pod: pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
			},
			Labels:       service.Labels,
			ReplicaCount: replicaCount,
			ServiceName:  service.Name,
		}
		test.Run(cs, ns, false)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: WITH STATEFULSETS: PASS\n"); err != nil {
			panic(err)
		}
	})
})

var _ = Describe("[ics-e2e] [node-drain] [with-pods] Dynamic Provisioning using statefulsets", func() {
	f := framework.NewDefaultFramework("ics-e2e-statefulsets")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs clientset.Interface
		ns *v1.Namespace
	)
	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
	})
	It("With statefulset: should create one pod and attach volume dynamically. Write and read to volume. Next drain the node where pod is attached and wait for the pod to come up on second node. Read the volume again.", func() {

		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		var replicaCount = int32(1)
		headlessService := testsuites.NewHeadlessService(cs, "ics-e2e-service-", ns.Name, "node-drain")
		service := headlessService.Create()
		defer headlessService.Cleanup()

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		pod := testsuites.PodDetails{
			Cmd: "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-5iops",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "20Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}

		test := testsuites.StatefulsetWithVolWRTest{
			Pod: pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
			},
			Labels:       service.Labels,
			ReplicaCount: replicaCount,
			ServiceName:  service.Name,
		}
		test.Run(cs, ns, true)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: CORDON AND DRAIN NODE: PASS\n"); err != nil {
			panic(err)
		}
	})
})

var _ = Describe("[ics-e2e] [resize] [pv] Dynamic Provisioning and resize pv", func() {
	f := framework.NewDefaultFramework("ics-e2e-pods")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs        clientset.Interface
		ns        *v1.Namespace
		secretKey string
	)

	secretKey = os.Getenv("E2E_SECRET_ENCRYPTION_KEY")
	if secretKey == "" {
		secretKey = defaultSecret
	}

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
	})

	It("with 5iops sc: should create a pvc & pv, pod resources, and resize the volume", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "ics-vol-5iops-",
						VolumeType:    "ibmc-vpc-block-5iops-tier",
						FSType:        "ext4",
						ClaimSize:     "260Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
			// ExpandVolSize is in Gi i.e, 2000Gi
			ExpandVolSizeG: 2000,
			ExpandedSize:   1900,
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5IOPS SC POD TEST AND RESIZE VOLUME: PASS\n"); err != nil {
			panic(err)
		}
	})
})

var _ = Describe("[ics-e2e] [snapshot] Dynamic Provisioning and Snapshot", func() {
	f := framework.NewDefaultFramework("ics-e2e-snap")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs          clientset.Interface
		snapshotrcs restclientset.Interface
		ns          *v1.Namespace
		secretKey   string
	)

	secretKey = os.Getenv("E2E_SECRET_ENCRYPTION_KEY")
	if secretKey == "" {
		secretKey = defaultSecret
	}

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		var err error
		snapshotrcs, err = restClient(testsuites.SnapshotAPIGroup, testsuites.APIVersionv1)
		if err != nil {
			Fail(fmt.Sprintf("could not get rest clientset: %v", err))
		}
	})

	It("should create a pod, write and read to it, take a volume snapshot, and create another pod from the snapshot", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pod := testsuites.PodDetails{
			// sync before taking a snapshot so that any cached data is written to the EBS volume
			Cmd:      "echo 'hello world' >> /mnt/test-1/data && grep 'hello world' /mnt/test-1/data && sync",
			CmdExits: false,
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-5iops-snap-",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "20Gi",
					ReclaimPolicy: &reclaimPolicy,
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		restoredPod1 := testsuites.PodDetails{
			Cmd: "grep 'hello world' /mnt/test-1/data && while true; do sleep 2; done",
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-5iops-snap-",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "20Gi",
					ReclaimPolicy: &reclaimPolicy,
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		test1 := testsuites.DynamicallyProvisionedVolumeSnapshotTest{
			Pod:         pod,
			RestoredPod: restoredPod1,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		By("VPC-BLK-CSI-TEST: SNAPSHOT CREATION | SAME CLAIM SIZE | DELETE SNAPSHOT")
		test1.Run(cs, snapshotrcs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: SNAPSHOT CREATION | SAME CLAIM SIZE | DELETE SNAPSHOT: PASS\n"); err != nil {
			panic(err)
		}

		restoredPod2 := testsuites.PodDetails{
			Cmd: "grep 'hello world' /mnt/test-1/data && while true; do sleep 2; done",
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-5iops-snap-",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "10Gi",
					ReclaimPolicy: &reclaimPolicy,
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		test2 := testsuites.DynamicallyProvisionedVolumeSnapshotTest{
			Pod:         pod,
			RestoredPod: restoredPod2,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		By("VPC-BLK-CSI-TEST: SNAPSHOT CREATION | CLAIM SIZE LESS | DELETE SNAPSHOT FOR WHICH SOURCE VOLUME DELETED")
		test2.VolumeSizeLess(cs, snapshotrcs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: SNAPSHOT CREATION | CLAIM SIZE LESS | DELETE SNAPSHOT FOR WHICH SOURCE VOLUME DELETED: PASS\n"); err != nil {
			panic(err)
		}
		restoredPod3 := testsuites.PodDetails{
			Cmd: "grep 'hello world' /mnt/test-1/data && while true; do sleep 2; done",
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vol-5iops-snap-",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "30Gi",
					ReclaimPolicy: &reclaimPolicy,
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		test3 := testsuites.DynamicallyProvisionedVolumeSnapshotTest{
			Pod:         pod,
			RestoredPod: restoredPod3,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		By("VPC-BLK-CSI-TEST: SNAPSHOT CREATION | CLAIM SIZE MORE | DELETE SNAPSHOT: PASS")
		test3.Run(cs, snapshotrcs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: SNAPSHOT CREATION | CLAIM SIZE MORE | DELETE SNAPSHOT: PASS\n"); err != nil {
			panic(err)
		}

		// Snapshot for unattached volume
		By("VPC-BLK-CSI-TEST: SNAPSHOT CREATION FOR UNATTACHED VOLUME MUST FAIL")
		test1.SnapShotForUnattached(cs, snapshotrcs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: SNAPSHOT CREATION FOR UNATTACHED VOLUME MUST FAIL: PASS\n"); err != nil {
			panic(err)
		}
	})
})

var _ = Describe("[ics-e2e] [sc] [with-deploy] Provisioning PVC with SDP profile", func() {
	f := framework.NewDefaultFramework("ics-e2e-deploy")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	condition := true // or some env check version
	var (
		cs          clientset.Interface
		snapshotrcs restclientset.Interface
		ns          *v1.Namespace
		secretKey   string
	)

	secretKey = os.Getenv("E2E_SECRET_ENCRYPTION_KEY")
	if secretKey == "" {
		secretKey = defaultSecret
	}

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		var err error
		version := ""
		deployment, err := cs.AppsV1().Deployments("kube-system").Get(context.TODO(), "ibm-vpc-block-csi-controller", metav1.GetOptions{})
		if err != nil {
			log.Printf("Error getting Deployment: %v", err)
			sts, err := cs.AppsV1().StatefulSets("kube-system").Get(context.TODO(), "ibm-vpc-block-csi-controller", metav1.GetOptions{})
			if err != nil {
				log.Fatalln("Error getting Sts and Deployment")
			}
			log.Println("STS Found")
			version = sts.ObjectMeta.Annotations["version"]
			fmt.Println("Addon version:", version)
		} else {
			log.Println("Deployment Found")
			// Extract version annotation
			version = deployment.ObjectMeta.Annotations["version"]
			fmt.Println("Addon version:", version)
		}

		//saving x.y version
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			majorMinor := fmt.Sprintf("%s.%s", parts[0], parts[1])
			fmt.Println("Major.Minor:", majorMinor)
			if majorMinor == "5.1" {
				condition = false
			}
		} else {
			fmt.Println("Version format is invalid")
		}

		snapshotrcs, err = restClient(testsuites.SnapshotAPIGroup, testsuites.APIVersionv1)
		if err != nil {
			Fail(fmt.Sprintf("could not get rest clientset: %v", err))
		}
	})

	// sc and pvc are according to doc, positive scenerio,  pvc creation will be successful
	It("with custom sc(iops=3000, throughput=1000, pvc size=1Gi): should create a pvc & pv, pod resources, write and read to volume", func() {
		if condition == false {
			Skip("Skipping because addon version is 5.1 and acadia profile is not supported for this version")
		}
		//create sc
		CreateSDPStorageClass("sdp-test-sc", "3000", "1000", cs)
		// Defer the deletion of the StorageClass object.
		defer func() {
			if err := cs.StorageV1().StorageClasses().Delete(context.Background(), "sdp-test-sc", metav1.DeleteOptions{}); err != nil {
				panic(err)
			}
		}()

		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		// For Custom SC PVC name and Secret name should be same and in same NS
		secret := testsuites.NewSecret(cs, "sdp-test-pvc", ns.Name, "800", "e2e test",
			"false", secretKey, "vpc.block.csi.ibm.io")
		secret.Create()
		defer secret.Cleanup()
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "sdp-test-pvc",
						VolumeType:    "sdp-test-sc",
						FSType:        "ext4",
						ClaimSize:     "1Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: CUSTOM SC POD With SDP PROFILE PVC(1Gi) TEST: PASS\n"); err != nil {
			panic(err)
		}

		test1 := testsuites.DynamicallyProvisionedResizeVolumeTest{
			Pods: pods,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
			// ExpandVolSize is in Gi i.e, 10Gi
			ExpandVolSizeG: 10,
			ExpandedSize:   9,
		}
		test1.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: CUSTOM SC with SDP PROFILE PVC RESIZE VOLUME: PASS\n"); err != nil {
			panic(err)
		}
	})

	// sc and pvc are according to doc, positive scenerio,  pvc creation will be successful
	It("with custom sc(iops=4000, throughput=8000, pvc size=90Gi): should create a pvc & pv, pod resources, write and read to volume", func() {
		if condition == false {
			Skip("Skipping because addon version is 5.1 and acadia profile is not supported for this version")
		}
		//create sc
		CreateSDPStorageClass("sdp-test-sc", "4000", "8000", cs)
		// Defer the deletion of the StorageClass object.
		defer func() {
			if err := cs.StorageV1().StorageClasses().Delete(context.Background(), "sdp-test-sc", metav1.DeleteOptions{}); err != nil {
				panic(err)
			}
		}()

		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		// For Custom SC PVC name and Secret name should be same and in same NS
		secret := testsuites.NewSecret(cs, "sdp-test-pvc", ns.Name, "800", "e2e test",
			"false", secretKey, "vpc.block.csi.ibm.io")
		secret.Create()
		defer secret.Cleanup()
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "sdp-test-pvc",
						VolumeType:    "sdp-test-sc",
						FSType:        "ext4",
						ClaimSize:     "90Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: CUSTOM SC POD with SDP Profile PVC(90Gi) TEST: PASS\n"); err != nil {
			panic(err)
		}
	})

	// sc and pvc are not according to doc, negative scenerio, pvc creation will fail
	It("with custom sc(iops=4000, throughput=2000, pvc size=10Gi): should create a pvc & pv, pod resources, write and read to volume", func() {
		if condition == false {
			Skip("Skipping because addon version is 5.1 and acadia profile is not supported for this version")
		}
		//create sc
		CreateSDPStorageClass("sdp-test-sc", "4000", "2000", cs)
		// Defer the deletion of the StorageClass object.
		defer func() {
			if err := cs.StorageV1().StorageClasses().Delete(context.Background(), "sdp-test-sc", metav1.DeleteOptions{}); err != nil {
				panic(err)
			}
		}()

		// create pvc
		CreateSDPPVC("sdp-test-pvc", "sdp-test-sc", ns.Name, 4000, 2000, "10Gi", cs)
		// Defer the deletion of the PVC object.
		defer func() {
			if err := cs.CoreV1().PersistentVolumeClaims(ns.Name).Delete(context.Background(), "sdp-test-pvc", metav1.DeleteOptions{}); err != nil {
				panic(err)
			}
		}()

		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()

		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: CUSTOM SC with SDP Profile POD TEST - Negative Case : PASS\n"); err != nil {
			panic(err)
		}
	})

	// sc and pvc are according to doc, positive scenerio,  pvc creation will be successful
	It("with custom sc(iops=4000, throughput=2000, pvc size=25Gi): should create a pvc & pv, pod resources, create snapshot and restore it", func() {
		if condition == false {
			Skip("Skipping because addon version is 5.1 and acadia profile is not supported for this version")
		}
		//create sc
		CreateSDPStorageClass("sdp-test-sc", "4000", "2000", cs)
		// Defer the deletion of the StorageClass object.
		defer func() {
			if err := cs.StorageV1().StorageClasses().Delete(context.Background(), "sdp-test-sc", metav1.DeleteOptions{}); err != nil {
				panic(err)
			}
		}()

		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, labelerr := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		if labelerr != nil {
			panic(labelerr)
		}
		// For Custom SC PVC name and Secret name should be same and in same NS
		secret := testsuites.NewSecret(cs, "sdp-test-pvc", ns.Name, "800", "e2e test",
			"false", secretKey, "vpc.block.csi.ibm.io")
		secret.Create()
		defer secret.Cleanup()
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		pods := []testsuites.PodDetails{
			{
				Cmd:      "echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done",
				CmdExits: false,
				Volumes: []testsuites.VolumeDetails{
					{
						PVCName:       "sdp-test-pvc",
						VolumeType:    "sdp-test-sc",
						FSType:        "ext4",
						ClaimSize:     "25Gi",
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
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: CUSTOM SC POD TEST: PASS\n"); err != nil {
			panic(err)
		}

		//creating snapshot

		pod := testsuites.PodDetails{
			// sync before taking a snapshot so that any cached data is written to the EBS volume
			Cmd:      "echo 'hello world' >> /mnt/test-1/data && grep 'hello world' /mnt/test-1/data && sync",
			CmdExits: false,
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "sdp-test-pvc-1",
					VolumeType:    "sdp-test-sc",
					FSType:        "ext4",
					ClaimSize:     "25Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		restoredPod1 := testsuites.PodDetails{
			Cmd: "grep 'hello world' /mnt/test-1/data && while true; do sleep 2; done",
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "sdp-test-pvc-2",
					VolumeType:    "sdp-test-sc",
					FSType:        "ext4",
					ClaimSize:     "25Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		test1 := testsuites.DynamicallyProvisionedVolumeSnapshotTest{
			Pod:         pod,
			RestoredPod: restoredPod1,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}

		test1.Run(cs, snapshotrcs, ns)
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: Custom SC with SDP Profile SNAPSHOT CREATION | RESTORE | SAME CLAIM SIZE | DELETE SNAPSHOT: PASS\n"); err != nil {
			panic(err)
		}
	})

})

func restClient(group string, version string) (restclientset.Interface, error) {
	// setup rest client
	config, err := framework.LoadConfig()
	if err != nil {
		Fail(fmt.Sprintf("could not load config: %v", err))
	}
	gv := schema.GroupVersion{Group: group, Version: version}
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: serializer.NewCodecFactory(runtime.NewScheme())}
	return restclientset.RESTClientFor(config)
}
