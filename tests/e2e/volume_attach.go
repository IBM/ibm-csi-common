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
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
)

const (
	addonConfigMap = "addon-vpc-block-csi-driver-configmap"
	configMapNs    = "kube-system"
	customSCName   = "custom-sc"
)

var _ = Describe("[ics-e2e] [volume-attachment-limit] [config] [3-volumes]", func() {
	f := framework.NewDefaultFramework("ics-e2e-pods")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs        clientset.Interface
		ns        *corev1.Namespace
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

	It("Configmap Parameter is set to 3. Verify volume attachment scenarios", func() {
		By("VOLUME ATTACHMENT WITH 3 VOLUMES")
		CreateStorageClass(customSCName, cs)
		// Defer the deletion of the StorageClass object.
		defer func() {
			if err := cs.StorageV1().StorageClasses().Delete(context.Background(), customSCName, metav1.DeleteOptions{}); err != nil {
				panic(err)
			}
		}()
		for i := 0; i < 4; i++ {
			CreatePVC("pvc-"+strconv.Itoa(i), ns.Name, cs)
		}
		UpdateVolumeAttachmentLimit(cs, "3")
		time.Sleep(650 * time.Second)
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
								Image:   "us.icr.io/armada-master/agnhost:2.52",
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
						Volumes: []corev1.Volume{
							{
								Name: "volume-1",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-0",
									},
								},
							},
							{
								Name: "volume-2",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
							{
								Name: "volume-3",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-2",
									},
								},
							},
						},
					},
				},
			},
		}
		statefulSet, err := cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

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
		err = cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: VOLUME_ATTACHMENT_LIMIT SET TO 3 && ATTACHED VOLUMES 3: PASS\n"); err != nil {
			panic(err)
		}

		By("VOLUME ATTACHMENT WITH 4 VOLUMES")
		statefulSet2 := "test-statefulset-two"
		replicas = int32(1)

		// Define StatefulSet
		statefulSet = &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulSet2,
				Namespace: ns.Name,
			},
			Spec: apps.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: statefulSet2,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": statefulSet2},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": statefulSet2},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "example-container",
								Image:   "us.icr.io/armada-master/agnhost:2.52",
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
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "volume-1",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-0",
									},
								},
							},
							{
								Name: "volume-2",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
							{
								Name: "volume-3",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-2",
									},
								},
							},
							{
								Name: "volume-4",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-3",
									},
								},
							},
						},
					},
				},
			},
		}
		// Create the StatefulSet
		statefulSet, err = cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Wait for the StatefulSet to be ready
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			ss, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSet2, metav1.GetOptions{})
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
		_, err = cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSet2, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: VOLUME_ATTACHMENT_LIMIT SET TO 3 && ATTACHING 4 VOLUMES MUST FAIL: PASS\n"); err != nil {
			panic(err)
		}
	})

})

var _ = Describe("[ics-e2e] [volume-attachment-limit] [default] [12-volumes]", func() {
	f := framework.NewDefaultFramework("ics-e2e-pods")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	var (
		cs        clientset.Interface
		ns        *corev1.Namespace
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

	It("Verify volume attachment without any change in configmap", func() {
		By("DEFAULT VOLUME ATTACHMENT WITH 12 VOLUMES")
		CreateStorageClass(customSCName, cs)
		// Defer the deletion of the StorageClass object.
		defer func() {
			if err := cs.StorageV1().StorageClasses().Delete(context.Background(), customSCName, metav1.DeleteOptions{}); err != nil {
				panic(err)
			}
		}()
		for i := 0; i < 13; i++ {
			CreatePVC("pvc-"+strconv.Itoa(i), ns.Name, cs)
		}
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
								Image:   "us.icr.io/armada-master/agnhost:2.52",
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
						Volumes: []corev1.Volume{
							{
								Name: "volume-1",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-0",
									},
								},
							},
							{
								Name: "volume-2",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
							{
								Name: "volume-3",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-2",
									},
								},
							},
							{
								Name: "volume-4",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-3",
									},
								},
							},
							{
								Name: "volume-5",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-4",
									},
								},
							},
							{
								Name: "volume-6",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-5",
									},
								},
							},
							{
								Name: "volume-7",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-6",
									},
								},
							},
							{
								Name: "volume-8",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-7",
									},
								},
							},
							{
								Name: "volume-9",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-8",
									},
								},
							},
							{
								Name: "volume-10",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-9",
									},
								},
							},
							{
								Name: "volume-11",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-10",
									},
								},
							},
							{
								Name: "volume-12",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-11",
									},
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
		err = cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: DEFAULT VOLUME ATTACHMENT WITH 12 VOLUMES: PASS\n"); err != nil {
			panic(err)
		}

		By("DEFAULT VOLUME ATTACHMENT WITH 13 VOLUMES")

		statefulSetTwo := "test-statefulset-two"
		replicas = int32(1)

		// Define StatefulSet
		statefulSet = &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulSetTwo,
				Namespace: ns.Name,
			},
			Spec: apps.StatefulSetSpec{
				Replicas:    &replicas,
				ServiceName: statefulSetTwo,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": statefulSetTwo},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": statefulSetTwo},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "example-container",
								Image:   "us.icr.io/armada-master/agnhost:2.52",
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
						Volumes: []corev1.Volume{
							{
								Name: "volume-1",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-0",
									},
								},
							},
							{
								Name: "volume-2",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
							{
								Name: "volume-3",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-2",
									},
								},
							},
							{
								Name: "volume-4",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-3",
									},
								},
							},
							{
								Name: "volume-5",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-4",
									},
								},
							},
							{
								Name: "volume-6",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-5",
									},
								},
							},
							{
								Name: "volume-7",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-6",
									},
								},
							},
							{
								Name: "volume-8",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-7",
									},
								},
							},
							{
								Name: "volume-9",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-8",
									},
								},
							},
							{
								Name: "volume-10",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-9",
									},
								},
							},
							{
								Name: "volume-11",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-10",
									},
								},
							},
							{
								Name: "volume-12",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-11",
									},
								},
							},
							{
								Name: "volume-13",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-12",
									},
								},
							},
						},
					},
				},
			},
		}
		// Create the StatefulSet
		statefulSet, err = cs.AppsV1().StatefulSets(ns.Name).Create(context.TODO(), statefulSet, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Wait for the StatefulSet to be ready
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			ss, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetTwo, metav1.GetOptions{})
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
		_, err = cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), statefulSetTwo, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = cs.AppsV1().StatefulSets(ns.Name).Delete(context.TODO(), statefulSet.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
		if _, err = fpointer.WriteString("VPC-BLK-CSI-TEST: DEFAULT VOLUME ATTACHMENT WITH 13 VOLUMES MUST FAIL: PASS\n"); err != nil {
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

func CreatePVC(pvcName string, namespace string, cs clientset.Interface) {
	customSCName := "custom-sc"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &customSCName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("10Gi"),
					//corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}

	// Create the PVC
	_, err := cs.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	err = wait.PollImmediate(5*time.Second, 60*time.Second, func() (bool, error) {
		updatedPVC, err := cs.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return updatedPVC.Status.Phase == corev1.ClaimBound, nil
	})
	if err != nil {
		panic(err)
	}
}

func CreateStorageClass(scName string, cs clientset.Interface) {
	// Create a StorageClass object.
	var zone = os.Getenv("E2E_ZONE")
	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: scName,
		},
		Provisioner: "vpc.block.csi.ibm.io",
		Parameters: map[string]string{
			"profile":                   "5iops-tier",
			"zone":                      zone,
			"csi.storage.k8s.io/fstype": "ext4",
			"billingType":               "hourly",
		},
	}
	// Create the StorageClass object.
	_, err = cs.StorageV1().StorageClasses().Create(context.Background(), storageClass, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
}
