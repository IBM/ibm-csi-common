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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
	"os"
	"time"
)

var _ = Describe("[ics-e2e] [volumeattachment] [with-sts]", func() {
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
		fpointer, err = os.OpenFile(testResultFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer fpointer.Close()
		statefulSetName := "test-statefulset"
		replicas := int32(1)

		// Define StatefulSet
		statefulSet := &apps.StatefulSet{
			ObjectMeta: meta.ObjectMeta{
				Name:      statefulSetName,
				Namespace: ns.Name,
			},
			Spec: apps.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: statefulSetName,
				Selector: &meta.LabelSelector{
					MatchLabels: map[string]string{"app": statefulSetName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: meta.ObjectMeta{
						Labels: map[string]string{"app": statefulSetName},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "example-container",
								Image:   "nginx",
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", "echo 'hello world' > /data/volume-1/data && while true; do sleep 2; done"},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "volume-1",
										MountPath: "/data/volume-1",
									},
								        {
										Name:      "volume-2",
										MountPath: "/data/volume-2",
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: meta.ObjectMeta{
							Name: "volume-1",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
                                        {
                                                ObjectMeta: meta.ObjectMeta{
                                                        Name: "volume-2",
                                                },
                                                Spec: corev1.PersistentVolumeClaimSpec{
                                                        AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
                                                        Resources: corev1.ResourceRequirements{
                                                                Requests: corev1.ResourceList{
                                                                        corev1.ResourceStorage: resource.MustParse("1Gi"),
                                                                },
                                                        },
                                                },
                                        },
				},
			},
		}
		// Create the StatefulSet
		statefulSet, err := cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, meta.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, meta.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}()

		// Wait for the StatefulSet to be ready
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			ss, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, meta.GetOptions{})
			if err != nil {
				return false, err
			}
			if ss.Status.ReadyReplicas == replicas {
				return true, nil
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Verify the StatefulSet exists
		_, err = cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, meta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5iops SC POD Test: PASS\n"); err != nil {
			panic(err)
		}
	})

})
