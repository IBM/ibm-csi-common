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
package snapshote2e

import (
	"context"
	"fmt"
	"os"

	"github.com/IBM/ibm-csi-common/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
)

const defaultSecret = ""

var testResultFile = os.Getenv("E2E_TEST_RESULT")
var err error
var fpointer *os.File

var _ = Describe("[ics-e2e] [snapshot] Dynamic Provisioning and Snapshot", func() {
	f := framework.NewDefaultFramework("ics-e2e-snap")

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
		fpointer, err = os.OpenFile(testResultFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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
