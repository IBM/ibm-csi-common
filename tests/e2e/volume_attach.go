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

const (
	addonConfigMap = "addon-vpc-block-csi-driver-configmap"
	configMapNs    = "kube-system"
)

var _ = Describe("[ics-e2e] [volume-attachment-limit] [default-12-volumes]", func() {
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulSetName,
				Namespace: ns.Name,
			},
			Spec: apps.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: statefulSetName,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": statefulSetName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
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
									{
										Name:      "volume-3",
										MountPath: "/data/volume-3",
									},
									{
										Name:      "volume-4",
										MountPath: "/data/volume-4",
									},
									{
										Name:      "volume-5",
										MountPath: "/data/volume-5",
									},
									{
										Name:      "volume-6",
										MountPath: "/data/volume-6",
									},
									{
										Name:      "volume-7",
										MountPath: "/data/volume-7",
									},
									{
										Name:      "volume-8",
										MountPath: "/data/volume-8",
									},
									{
										Name:      "volume-9",
										MountPath: "/data/volume-9",
									},
									{
										Name:      "volume-10",
										MountPath: "/data/volume-10",
									},
									{
										Name:      "volume-11",
										MountPath: "/data/volume-11",
									},
									{
										Name:      "volume-12",
										MountPath: "/data/volume-12",
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
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
						ObjectMeta: metav1.ObjectMeta{
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
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-3",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-4",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-5",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-6",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-7",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-8",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-9",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-10",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-11",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-12",
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
		statefulSet, err := cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}()

		// Wait for the StatefulSet to be ready
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			ss, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
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
		_, err = cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5iops SC POD Test: PASS\n"); err != nil {
			panic(err)
		}
	})

})
var _ = Describe("[ics-e2e] [volume-attachment-limit] [default-13-volumes]", func() {
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulSetName,
				Namespace: ns.Name,
			},
			Spec: apps.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: statefulSetName,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": statefulSetName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
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
									{
										Name:      "volume-3",
										MountPath: "/data/volume-3",
									},
									{
										Name:      "volume-4",
										MountPath: "/data/volume-4",
									},
									{
										Name:      "volume-5",
										MountPath: "/data/volume-5",
									},
									{
										Name:      "volume-6",
										MountPath: "/data/volume-6",
									},
									{
										Name:      "volume-7",
										MountPath: "/data/volume-7",
									},
									{
										Name:      "volume-8",
										MountPath: "/data/volume-8",
									},
									{
										Name:      "volume-9",
										MountPath: "/data/volume-9",
									},
									{
										Name:      "volume-10",
										MountPath: "/data/volume-10",
									},
									{
										Name:      "volume-11",
										MountPath: "/data/volume-11",
									},
									{
										Name:      "volume-12",
										MountPath: "/data/volume-12",
									},
									{
										Name:      "volume-13",
										MountPath: "/data/volume-13",
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
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
						ObjectMeta: metav1.ObjectMeta{
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
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-3",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-4",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-5",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-6",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-7",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-8",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-9",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-10",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-11",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-12",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-13",
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
		statefulSet, err := cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}()

		// Wait for the StatefulSet to be ready
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			ss, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if ss.Status.ReadyReplicas == replicas {
				return true, nil
			}
			return false, nil
		})
		Expect(err).To(HaveOccurred())

		// Verify the StatefulSet exists
		_, err = cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5iops SC POD Test: PASS\n"); err != nil {
			panic(err)
		}
	})

})

var _ = Describe("[ics-e2e] [volume-attachment-limit] [config-3-volumes]", func() {
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
		UpdateVolumeAttachmentLimit(cs, "3")
		time.Sleep(50 * time.Second)
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulSetName,
				Namespace: ns.Name,
			},
			Spec: apps.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: statefulSetName,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": statefulSetName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
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
									{
										Name:      "volume-3",
										MountPath: "/data/volume-3",
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
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
						ObjectMeta: metav1.ObjectMeta{
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
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-3",
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
		statefulSet, err := cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}()

		// Wait for the StatefulSet to be ready
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			ss, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
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
		_, err = cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5iops SC POD Test: PASS\n"); err != nil {
			panic(err)
		}
	})

})

var _ = Describe("[ics-e2e] [volume-attachment-limit] [config-13-volumes]", func() {
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
		UpdateVolumeAttachmentLimit(cs, "13")
		time.Sleep(50 * time.Second)
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulSetName,
				Namespace: ns.Name,
			},
			Spec: apps.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: statefulSetName,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": statefulSetName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
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
									{
										Name:      "volume-3",
										MountPath: "/data/volume-3",
									},
									{
										Name:      "volume-4",
										MountPath: "/data/volume-4",
									},
									{
										Name:      "volume-5",
										MountPath: "/data/volume-5",
									},
									{
										Name:      "volume-6",
										MountPath: "/data/volume-6",
									},
									{
										Name:      "volume-7",
										MountPath: "/data/volume-7",
									},
									{
										Name:      "volume-8",
										MountPath: "/data/volume-8",
									},
									{
										Name:      "volume-9",
										MountPath: "/data/volume-9",
									},
									{
										Name:      "volume-10",
										MountPath: "/data/volume-10",
									},
									{
										Name:      "volume-11",
										MountPath: "/data/volume-11",
									},
									{
										Name:      "volume-12",
										MountPath: "/data/volume-12",
									},
									{
										Name:      "volume-13",
										MountPath: "/data/volume-13",
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
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
						ObjectMeta: metav1.ObjectMeta{
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
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-3",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-4",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-5",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-6",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-7",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-8",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-9",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-10",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-11",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-12",
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
						ObjectMeta: metav1.ObjectMeta{
							Name: "volume-13",
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
		statefulSet, err := cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}()

		// Wait for the StatefulSet to be ready
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			ss, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
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
		_, err = cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: 5iops SC POD Test: PASS\n"); err != nil {
			panic(err)
		}
	})

})

func UpdateVolumeAttachmentLimit(client clientset.Interface, limit string) bool {
	iksConfigMap, err := client.CoreV1().ConfigMaps(configMapNs).Get(context.TODO(), addonConfigMap, metav1.GetOptions{})
	if err != nil {
		return false
	}
	iksConfigMap.Data["VolumeAttachmentLimit"] = limit
	_, err = client.CoreV1().ConfigMaps(configMapNs).Update(context.TODO(), iksConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return false
	}
	return true

}
