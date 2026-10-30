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
	"strconv"
	"strings"

	"github.com/IBM/ibm-csi-common/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
)

const vgsDriverVersion = "5.2"

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
		if driverVersion != vgsDriverVersion {
			Skip(fmt.Sprintf("VolumeGroupSnapshot tests require VPC Block CSI Driver %s; found %s", vgsDriverVersion, driverVersion))
		}
	})

	// patchNamespace applies the OpenShift / pod-security labels needed for
	// privileged pods in the test namespace.
	patchNamespace := func() {
		payload := `{"metadata": {"labels": {"security.openshift.io/scc.podSecurityLabelSync": "false","pod-security.kubernetes.io/enforce": "privileged"}}}`
		_, err := cs.CoreV1().Namespaces().Patch(context.TODO(), ns.Name, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		framework.ExpectNoError(err)
	}

	// twoVolumeTest returns a pre-populated test struct with two 20 GiB source volumes.
	twoVolumeTest := func() testsuites.DynamicallyProvisionedVolumeGroupSnapshotTest {
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		return testsuites.DynamicallyProvisionedVolumeGroupSnapshotTest{
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
	}

	// Report after framework cleanup as well as the scenario's own cleanup.
	ReportAfterEach(func(report SpecReport) {
		// Filtered-out VGS specs are reported by Ginkgo during the other suites.
		// They never ran and must not write result entries.
		if report.NumAttempts == 0 {
			return
		}
		status := "FAILED"
		if report.State.Is(ginkgotypes.SpecStatePassed) {
			status = "PASS"
		}
		if report.State.Is(ginkgotypes.SpecStateSkipped) {
			status = "SKIP"
		}
		resultFile, err := os.OpenFile(testResultFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer resultFile.Close()
		if _, err = resultFile.WriteString(vgsResultKey(report.LeafNodeText) + ": " + status + "\n"); err != nil {
			panic(err)
		}
	})

	// -------------------------------------------------------------------------
	// Positive scenarios
	// -------------------------------------------------------------------------

	It("[positive] should snapshot two PVCs together, restore both volumes, verify their data, and delete the group snapshot", func() {
		patchNamespace()
		test := twoVolumeTest()
		test.Run(cs, dynamicClient, ns)
	})

	It("[positive] [selector-isolation] should exclude an unselected third PVC from the group snapshot", func() {
		patchNamespace()
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		test := testsuites.DynamicallyProvisionedVolumeGroupSnapshotTest{
			Volumes: []testsuites.VolumeDetails{
				{PVCName: "ics-vgs-iso-1-", VolumeType: "ibmc-vpc-block-5iops-tier", FSType: "ext4", ClaimSize: "20Gi", ReclaimPolicy: &reclaimPolicy, MountOptions: []string{"rw"}},
				{PVCName: "ics-vgs-iso-2-", VolumeType: "ibmc-vpc-block-5iops-tier", FSType: "ext4", ClaimSize: "20Gi", ReclaimPolicy: &reclaimPolicy, MountOptions: []string{"rw"}},
				{PVCName: "ics-vgs-iso-3-", VolumeType: "ibmc-vpc-block-5iops-tier", FSType: "ext4", ClaimSize: "20Gi", ReclaimPolicy: &reclaimPolicy, MountOptions: []string{"rw"}},
			},
		}
		test.RunSelectorIsolation(cs, dynamicClient, ns)
	})

	It("[positive] [point-in-time] restored volumes must contain data from snapshot time, not data written after", func() {
		patchNamespace()
		test := twoVolumeTest()
		test.RunPointInTimeRestore(cs, dynamicClient, ns)
	})

	It("[positive] [restore-after-source-delete] should restore data after source PVCs are deleted", func() {
		patchNamespace()
		test := twoVolumeTest()
		test.RunRestoreAfterSourceDelete(cs, dynamicClient, ns)
	})

	It("[positive] [delete-while-creating] deleting before readiness must not leak snapshot resources", func() {
		patchNamespace()
		test := twoVolumeTest()
		test.RunDeleteWhileCreating(cs, dynamicClient, ns)
	})

	It("[positive] [delete-idempotency] a repeated Kubernetes delete must report NotFound after all snapshot resources are removed", func() {
		patchNamespace()
		test := twoVolumeTest()
		test.RunDeleteIdempotency(cs, dynamicClient, ns)
	})

	// -------------------------------------------------------------------------
	// Negative scenarios
	// -------------------------------------------------------------------------

	It("[negative] [no-pvcs] a VGS whose selector matches no PVCs must report an error and not become ready", func() {
		patchNamespace()
		test := testsuites.DynamicallyProvisionedVolumeGroupSnapshotTest{}
		test.RunNoPVCs(cs, dynamicClient, ns)
	})

	It("[negative] [unattached] a VGS over unattached PVCs must report an error and not become ready", func() {
		patchNamespace()
		test := twoVolumeTest()
		test.RunUnattachedVolumes(cs, dynamicClient, ns)
	})

	It("[negative] [invalid-class] a VGS referencing a non-existent class must report an error and not become ready", func() {
		patchNamespace()
		test := twoVolumeTest()
		test.RunInvalidClass(cs, dynamicClient, ns)
	})
})

func vgsResultKey(description string) string {
	for _, item := range []struct{ tag, result string }{
		{"[selector-isolation]", "SELECTOR ISOLATION"},
		{"[point-in-time]", "POINT IN TIME RESTORE"},
		{"[restore-after-source-delete]", "RESTORE AFTER SOURCE DELETE"},
		{"[delete-while-creating]", "DELETE WHILE CREATING"},
		{"[delete-idempotency]", "REPEATED KUBERNETES DELETE"},
		{"[no-pvcs]", "NEGATIVE NO PVCS"},
		{"[unattached]", "NEGATIVE UNATTACHED"},
		{"[invalid-class]", "NEGATIVE INVALID CLASS"},
	} {
		if strings.Contains(description, item.tag) {
			return "VPC-BLK-CSI-TEST: VOLUME GROUP SNAPSHOT " + item.result
		}
	}
	return "VPC-BLK-CSI-TEST: VOLUME GROUP SNAPSHOT CREATE | RESTORE | DELETE"
}

// vpcBlockCSIDriverMajorMinorVersion returns the "major.minor" version string
// of the IBM VPC Block CSI controller, reading the "version" annotation from
// either its Deployment or StatefulSet in kube-system.
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
	for _, part := range parts[:2] {
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return "", fmt.Errorf("VPC Block CSI controller has invalid version annotation %q", version)
		}
	}
	return strings.Join(parts[:2], "."), nil
}
