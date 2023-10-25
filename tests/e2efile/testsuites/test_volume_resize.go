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

	. "github.com/onsi/ginkgo/v2"
	clientset "k8s.io/client-go/kubernetes"
)

// DynamicallyProvisionedResizeVolumeTest will provision required StorageClass(es), PVC(s) and Pod(s)
// Waiting for the PV provisioner to create a new PV
// Update pvc storage size
// Waiting for new PVC and PV to be ready
// And finally attach pvc to the pod and wait for pod to be ready.
type DynamicallyProvisionedResizeVolumeTest struct {
	Pods           []PodDetails
	PodCheck       *PodExecCheck
	ExpandVolSizeG int64
	ExpandedSize   int64
}

func (t *DynamicallyProvisionedResizeVolumeTest) Run(client clientset.Interface, namespace *v1.Namespace) {
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
		pod, err := client.CoreV1().Pods(namespace.Name).Get(context.Background(), tpod.pod.Name, metav1.GetOptions{})
		pvcName := getClaimsForPod(pod)
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Get(context.Background(), pvcName[0], metav1.GetOptions{})
		By(fmt.Sprintf("Get pvc name: %v", pvc.Name))
		delta := resource.Quantity{}
		delta.Set(t.ExpandVolSizeG * 1024 * 1024 * 1024)
		pvc.Spec.Resources.Requests["storage"] = delta

		By("resizing the pvc")
		updatedPvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Update(context.Background(), pvc, metav1.UpdateOptions{})
		if err != nil {
			framework.ExpectNoError(err, fmt.Sprintf("fail to resize pvc(%s): %v", pvcName, err))
		}
		updatedSize := updatedPvc.Spec.Resources.Requests["storage"]

		By("checking the resizing PV result")
		error := WaitForPvToResize(client, namespace, updatedPvc.Spec.VolumeName, updatedSize, 5*time.Minute, 5*time.Second)
		framework.ExpectNoError(error)
		By("checking filesystem resize result")
		err = WaitForFileSystemToResize(tpod, t.ExpandedSize, 5*time.Minute, 5*time.Second)
		framework.ExpectNoError(err)
	}
}

// WaitForPvToResize waiting for pvc size to be resized to desired size
func WaitForPvToResize(c clientset.Interface, ns *v1.Namespace, pvName string, desiredSize resource.Quantity, timeout time.Duration, interval time.Duration) error {
	By(fmt.Sprintf("Waiting up to %v for pv in namespace %q to be complete", timeout, ns.Name))
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(interval) {
		newPv, _ := c.CoreV1().PersistentVolumes().Get(context.Background(), pvName, metav1.GetOptions{})
		newPvSize := newPv.Spec.Capacity["storage"]
		if desiredSize.Cmp(newPvSize) == 0 {
			By(fmt.Sprintf("Pv size is updated to %v", newPvSize.String()))
			return nil
		}
	}
	return fmt.Errorf("Gave up after waiting %v for pv %q to complete resizing", timeout, pvName)
}

// WaitForFileSystemToResize waiting for pvc size to be resized to desired size
func WaitForFileSystemToResize(t *TestPod, expectedOut int64, timeout time.Duration, interval time.Duration) error {
	By(fmt.Sprintf("Waiting up to %v for filesystem in namespace to be resized", timeout))
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(interval) {
		out, err := exec.Command("kubectl", fmt.Sprintf("--kubeconfig=%s", os.Getenv("KUBECONFIG")), "exec", t.pod.Name, fmt.Sprintf("--namespace=%s", t.pod.Namespace), "--", "/bin/sh", "-c", "df -h /mnt/test-1 | grep test-1 | awk -F' ' '{ print $1 }'").CombinedOutput()
		if err != nil {
			return err
		} else {
			s := strings.TrimSpace(string(out))
			updatedSize, _ := resource.ParseQuantity(s)
			if err != nil {
				fmt.Println(err)
				return err
			}
			if updatedSizeInt64, ok := updatedSize.AsInt64(); ok {
				fmt.Println("updatedSizeInt64", updatedSizeInt64)
				updatedSizeG := updatedSizeInt64 / (1000 * 1000 * 1000)
				fmt.Println(updatedSizeG, "GB")
				if updatedSizeG >= expectedOut {
					By(fmt.Sprintf("FileSystem size is updated to %vG", updatedSizeG))
					return nil
				}
			}
		}
	}
	return fmt.Errorf("Gave up after waiting %v for filesystem to complete resizing", timeout)
}

// Get PVC claims for the pod
func getClaimsForPod(pod *v1.Pod) []string {
	pvcClaimList := make([]string, 2)
	for i, volumespec := range pod.Spec.Volumes {
		if volumespec.PersistentVolumeClaim != nil {
			pvcClaimList[i] = volumespec.PersistentVolumeClaim.ClaimName
		}
	}
	return pvcClaimList
}
