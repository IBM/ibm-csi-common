/**
 * Copyright 2025 IBM Corp.
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
	"os"
	"strings"

	"github.com/IBM/ibm-csi-common/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
)

var _ = Describe("[ics-e2e] [vgs] Dynamic Provisioning and Volume Group Snapshot", func() {
	f := framework.NewDefaultFramework("ics-e2e-vgs")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged

	var (
		cs            clientset.Interface
		dynamicClient dynamic.Interface
		ns            *v1.Namespace
	)

	BeforeEach(func() {
		cs = f.ClientSet
		dynamicClient = f.DynamicClient
		ns = f.Namespace

		driverVersion, err := vpcBlockCSIDriverMajorMinorVersion(cs)
		framework.ExpectNoError(err)
		if driverVersion != "5.2" {
			Skip(fmt.Sprintf("VolumeGroupSnapshot tests require VPC Block CSI Driver 5.2; found %s", driverVersion))
		}
	})

	It("should snapshot two PVCs together, restore both volumes, verify their data, and delete the group snapshot", func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, err := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		framework.ExpectNoError(err)

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		test := testsuites.DynamicallyProvisionedVolumeGroupSnapshotTest{
			Volumes: []testsuites.VolumeDetails{
				{
					PVCName:       "ics-vgs-source-1-",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "20Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
				},
				{
					PVCName:       "ics-vgs-source-2-",
					VolumeType:    "ibmc-vpc-block-5iops-tier",
					FSType:        "ext4",
					ClaimSize:     "20Gi",
					ReclaimPolicy: &reclaimPolicy,
					MountOptions:  []string{"rw"},
				},
			},
		}

		test.Run(cs, dynamicClient, ns)

		resultFile, err := os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer resultFile.Close()
		if _, err = resultFile.WriteString("VPC-BLK-CSI-TEST: VOLUME GROUP SNAPSHOT CREATE | RESTORE | DELETE: PASS\n"); err != nil {
			panic(err)
		}
	})
})

func vpcBlockCSIDriverMajorMinorVersion(cs clientset.Interface) (string, error) {
	version := ""
	deployment, deploymentErr := cs.AppsV1().Deployments("kube-system").Get(context.Background(), "ibm-vpc-block-csi-controller", metav1.GetOptions{})
	if deploymentErr == nil {
		version = deployment.Annotations["version"]
	} else {
		statefulSet, statefulSetErr := cs.AppsV1().StatefulSets("kube-system").Get(context.Background(), "ibm-vpc-block-csi-controller", metav1.GetOptions{})
		if statefulSetErr != nil {
			return "", fmt.Errorf("failed to get VPC Block CSI controller deployment (%v) or statefulset (%v)", deploymentErr, statefulSetErr)
		}
		version = statefulSet.Annotations["version"]
	}

	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("VPC Block CSI controller has invalid version annotation %q", version)
	}
	return strings.Join(parts[:2], "."), nil
}
