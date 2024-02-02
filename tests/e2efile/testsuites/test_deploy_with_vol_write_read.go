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
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// DynamicallyProvisioneVolPodWRTest will provision required PVC and Deployment
// Testing if the Pod can write and read to mounted volumes
// Deleting a pod, and again testing if the Pod can write and read to mounted volumes
type DynamicallyProvisioneDeployWithVolWRTest struct {
	Pod          PodDetails
	PodCheck     *PodExecCheck
	ReplicaCount int32
}

func (t *DynamicallyProvisioneDeployWithVolWRTest) Run(client clientset.Interface, namespace *v1.Namespace) {
	tDeployment, cleanup := t.Pod.SetupDeployment(client, namespace, t.ReplicaCount)
	// defer must be called here for resources not get removed before using them
	for i := range cleanup {
		defer cleanup[i]()
	}

	By("deploying the deployment")
	tDeployment.Create()

	By("checking that the pod is running")
	tDeployment.WaitForPodReady()

	if t.PodCheck != nil {
		By("checking pod exec before pod delete")
		tDeployment.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString01)
	}
	By("deleting the pod for deployment")
	tDeployment.DeletePodAndWait()

	By("checking again that the pod is running")
	tDeployment.WaitForPodReady()

	if t.PodCheck != nil {
		By("checking pod exec after pod recreate")
		tDeployment.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString02)
	}
}

func (t *DynamicallyProvisioneDeployWithVolWRTest) RunMultiVol(client clientset.Interface, namespace *v1.Namespace) {
	tDeployment, cleanup := t.Pod.SetupDeploymentWithMultiVol(client, namespace)
	// defer must be called here for resources not get removed before using them
	for i := range cleanup {
		defer cleanup[i]()
	}

	By("deploying the deployment")
	tDeployment.Create()

	By("checking that the pod is running")
	tDeployment.WaitForPodReady()

	if t.PodCheck != nil {
		By("checking pod exec before pod delete")
		tDeployment.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString01)
	}
	By("deleting the pod for deployment: 1/1")
	tDeployment.DeletePodAndWait()

	By("checking again that the pod is running")
	tDeployment.WaitForPodReady()

	if t.PodCheck != nil {
		By("checking pod exec after pod recreate")
		tDeployment.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString02)
	}

	By("deleting the pod for deployment: 2/2")
	tDeployment.DeletePodAndWait()

	By("checking again that the pod is running")
	tDeployment.WaitForPodReady()
}
