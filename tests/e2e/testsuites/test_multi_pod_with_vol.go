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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

// DynamicallyProvisioneVolPodWRTest will provision required PVC and Deployment
// Testing if the Pod can write and read to mounted volumes
// Deleting a pod, and again testing if the Pod can write and read to mounted volumes
type DynamicallyProvisioneMultiPodWithVolTest struct {
	Pods     []PodDetails
	PodCheck *PodExecCheck
}

func (t *DynamicallyProvisioneMultiPodWithVolTest) podCreateAndWait(tpod *TestPod, cmdExits bool) {
	tpod.DumpLogOff()
	tpod.Create()
	defer tpod.Cleanup()

	if !cmdExits {
		tpod.WaitForRunningSlow()
		if t.PodCheck != nil {
			tpod.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString01)
		}
	} else {
		tpod.WaitForSuccess()
	}
}

func (t *DynamicallyProvisioneMultiPodWithVolTest) RunSync(client clientset.Interface, namespace *v1.Namespace) {
	var ix, maxCount int
	var nodeName []string

	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		framework.Logf("failed to get Nodes list")
	} else {
		for _, node := range nodes.Items {
			if !node.Spec.Unschedulable {
				for _, condition := range node.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "True" {
						nodeName = append(nodeName, node.Name)
					}
				}
			}
		}
		maxCount = len(nodeName)
	}

	for n, pod := range t.Pods {
		name := fmt.Sprintf("ics-e2e-tester-%d-", n+1)
		tpod, cleanup := pod.SetupWithPVC(client, namespace, name)
		// defer must be called here for resources not get removed before using them
		for i := range cleanup {
			defer cleanup[i]()
		}

		By(fmt.Sprintf("deploying the pod [%s]", name))

		if maxCount > 1 {
			if ix >= maxCount {
				ix = 0
			}
			framework.Logf("POD host node: [%s]", nodeName[ix])
			tpod.SetNodeSelector(map[string]string{"kubernetes.io/hostname": nodeName[ix]})
			ix = ix + 1
		}

		t.podCreateAndWait(tpod, pod.CmdExits)
	}
}

func (t *DynamicallyProvisioneMultiPodWithVolTest) RunAsync(client clientset.Interface, namespace *v1.Namespace) {
	var wg sync.WaitGroup
	var ix, maxCount int
	var nodeName []string

	wg.Add(len(t.Pods))
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		framework.Logf("failed to get Nodes list")
	} else {
		for _, node := range nodes.Items {
			if !node.Spec.Unschedulable {
				for _, condition := range node.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "True" {
						nodeName = append(nodeName, node.Name)
					}
				}
			}
		}
		maxCount = len(nodeName)
	}

	for n, pod := range t.Pods {
		name := fmt.Sprintf("ics-e2e-tester-%d-", n+1)
		tpod, cleanup := pod.SetupWithPVC(client, namespace, name)
		// defer must be called here for resources not get removed before using them
		for i := range cleanup {
			defer cleanup[i]()
		}

		By(fmt.Sprintf("deploying the pod [%s]", name))

		if maxCount > 1 {
			if ix >= maxCount {
				ix = 0
			}
			framework.Logf("POD host node: [%s]", nodeName[ix])
			tpod.SetNodeSelector(map[string]string{"kubernetes.io/hostname": nodeName[ix]})
			ix = ix + 1
		}

		tpod.DumpLogOff()
		//tpod.DumpDbgInfoOff()
		tpod.Create()
		// defer tpod.Cleanup()
		if !pod.CmdExits {
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				defer tpod.Cleanup()
				tpod.WaitForRunningSlow()
				if t.PodCheck != nil {
					By("checking pod exec")
					tpod.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString01)
				}
			}()
		} else {
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				defer tpod.Cleanup()
				tpod.WaitForSuccess()
			}()
		}
	}

	wg.Wait()
}
