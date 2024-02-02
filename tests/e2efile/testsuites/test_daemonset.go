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

//Daemonset will provision required PVC and pods
// Testing if the Pod can write and read to mounted volumes
//Drain a node and again testing if the Pod can write and read to mounted volumes

type DaemonsetWithVolWRTest struct {
	Pod          PodDetails
	PodCheck     *PodExecCheck
	Labels       map[string]string
	ReplicaCount int32
	ServiceName  string
}

func (t *DaemonsetWithVolWRTest) Run(client clientset.Interface, namespace *v1.Namespace, drainNode bool) {
	tDaemonset, cleanup := t.Pod.SetupDaemonset(client, namespace, t.ServiceName, t.Labels)
	// defer must be called here for resources not get removed before using them
	for i := range cleanup {
		defer cleanup[i]()
	}
	By("deploying the Daemonset")
	tDaemonset.Create()

	By("checking that the pod(s) is/are running")
	tDaemonset.WaitForPodReady()

	if t.PodCheck != nil {
		By("checking pod exec before pod delete")
		tDaemonset.Exec(t.PodCheck.Cmd, t.PodCheck.ExpectedString01)
	}
}
