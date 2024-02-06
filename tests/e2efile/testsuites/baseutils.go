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
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/IBM/vpc-beta-go-sdk/vpcbetav1"
	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapshotclientset "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	framework "k8s.io/kubernetes/test/e2e/framework"
	k8sDevStat "k8s.io/kubernetes/test/e2e/framework/daemonset"
	k8sDevDep "k8s.io/kubernetes/test/e2e/framework/deployment"
	k8sDevPod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2eoutput "k8s.io/kubernetes/test/e2e/framework/pod/output"
	k8sDevPV "k8s.io/kubernetes/test/e2e/framework/pv"
	imageutils "k8s.io/kubernetes/test/utils/image"
)

type TestSecret struct {
	client     clientset.Interface
	name       string
	namespace  string
	iops       string
	tags       string
	encrypt    string
	encryptKey string
	stype      string
	secret     *v1.Secret
}
type TestHeadlessService struct {
	client             clientset.Interface
	name               string
	namespace          string
	service            *v1.Service
	labelsAndSelectors string
}

type TestStatefulsets struct {
	client      clientset.Interface
	namespace   *v1.Namespace
	statefulset *apps.StatefulSet
	podName     []string
}

type TestDaemonsets struct {
	client    clientset.Interface
	namespace *v1.Namespace
	daemonset *apps.DaemonSet
	podName   []string
}

func NewSecret(c clientset.Interface, name, ns, iops, tags, encrypt, encryptKey, stype string) *TestSecret {
	return &TestSecret{
		client:     c,
		name:       name,
		namespace:  ns,
		iops:       iops,
		tags:       tags,
		encrypt:    encrypt,
		encryptKey: encryptKey,
		stype:      stype,
	}
}

func NewHeadlessService(c clientset.Interface, name, namespace, labelSelctors string) *TestHeadlessService {
	return &TestHeadlessService{
		client:             c,
		name:               name,
		namespace:          namespace,
		labelsAndSelectors: labelSelctors,
	}
}

func (t *TestPersistentVolumeClaim) NewTestStatefulset(c clientset.Interface, ns *v1.Namespace, servicename, command, storageClassName, volumeName, mountPath string, labels map[string]string, replicaCount int32) *TestStatefulsets {
	pvcTemplate := generatePVC(volumeName, t.namespace.Name, storageClassName, t.claimSize, t.accessMode, t.volumeMode, t.dataSource)
	generateName := "ics-e2e-tester-"
	return &TestStatefulsets{
		client:    c,
		namespace: ns,
		statefulset: &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
			},
			Spec: apps.StatefulSetSpec{
				Replicas: &replicaCount,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				ServiceName: servicename,
				VolumeClaimTemplates: []v1.PersistentVolumeClaim{
					*pvcTemplate,
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:    "statefulset",
								Image:   imageutils.GetE2EImage(imageutils.Nginx),
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", command},
								Ports: []v1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: int32(8080),
									},
								},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      volumeName,
										MountPath: mountPath,
									},
								},
							},
						},
					},
				},
			},
		},
	}

}

func (t *TestPersistentVolumeClaim) NewTestDaemonset(c clientset.Interface, ns *v1.Namespace, pvc *v1.PersistentVolumeClaim, command, storageClassName, volumeName, mountPath string, labels map[string]string) *TestDaemonsets {
	generateName := "ics-e2e-tester-"
	return &TestDaemonsets{
		client:    c,
		namespace: ns,
		daemonset: &apps.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
			},
			Spec: apps.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:    "daemonset",
								Image:   imageutils.GetE2EImage(imageutils.Nginx),
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", command},
								Ports: []v1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: int32(8080),
									},
								},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      volumeName,
										MountPath: mountPath,
									},
								},
							},
						},
						Volumes: []v1.Volume{
							{
								Name: volumeName,
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: pvc.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

}

func (t *TestDaemonsets) Logs() {
	for _, podname := range t.podName {
		body, err := podLogs(t.client, podname, t.namespace.Name)
		if err != nil {
			framework.Logf("Error getting logs for pod %s: %v", podname, err)
		} else {
			framework.Logf("Pod %s has the following logs: %s", podname, body)
		}
	}
}

func (t *TestDaemonsets) Create() {
	var err error
	t.daemonset, err = t.client.AppsV1().DaemonSets(t.namespace.Name).Create(context.Background(), t.daemonset, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	By("DaemonSet checking: Slpeeing for 2 mins to make sure pods are up and running ")
	time.Sleep(2 * time.Minute)
	present, err := k8sDevStat.CheckPresentOnNodes(t.client, t.daemonset, t.namespace.Name, 1)
	if !present {
		framework.Logf("CheckPresentOnNodes on failed")
	}
}

func (t *TestDaemonsets) WaitForPodReady() {
	var err error
	By("DaemonSet running: Checking if All Pods are running ")

	for _, podname := range t.podName {
		err = k8sDevPod.WaitForPodCondition(t.client, t.namespace.Name, podname, failedConditionDescription, slowPodStartTimeout, podRunningCondition)
		framework.ExpectNoError(err)
	}
}

func (t *TestDaemonsets) Exec(command []string, expectedString string) {
	for _, podname := range t.podName {
		By("DaemonSet Exec: executing cmd in pod")
		_, err := e2eoutput.LookForStringInPodExec(t.namespace.Name, podname, command, expectedString, execTimeout)
		framework.ExpectNoError(err)
	}
}

func (t *TestDaemonsets) Cleanup() {
	By("DaemonSets Cleanup: deleting DaemonSet")
	e2eoutput.DumpDebugInfo(t.client, t.namespace.Name)
	t.Logs()
	framework.Logf("deleting Daemonset %q/%q", t.namespace.Name, t.daemonset.Name)
	err := t.client.AppsV1().DaemonSets(t.namespace.Name).Delete(context.Background(), t.daemonset.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (h *TestHeadlessService) Create() v1.Service {
	var err error
	selectorValue := fmt.Sprintf("%s-%d", h.labelsAndSelectors, rand.Int())
	By(fmt.Sprintf("creating HeadlessService under ns:%q", h.namespace))
	headlessService := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    h.namespace,
			GenerateName: h.name,
			Labels: map[string]string{
				"app": selectorValue,
			},
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{Port: 80, Name: "http", Protocol: "TCP"},
			},
			Selector: map[string]string{
				"app": h.labelsAndSelectors,
			},
		},
	}
	h.service, err = h.client.CoreV1().Services(h.namespace).Create(context.Background(), headlessService, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	fmt.Println("HeadlessService Label ", h.service.Labels)
	return *h.service
}

func (h *TestHeadlessService) Cleanup() {
	framework.Logf("deleting headless service  %s", h.service.Name)
	err := h.client.CoreV1().Services(h.service.Namespace).Delete(context.Background(), h.service.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (t *TestStatefulsets) Logs() {
	for _, podname := range t.podName {
		body, err := podLogs(t.client, podname, t.namespace.Name)
		if err != nil {
			framework.Logf("Error getting logs for pod %s: %v", podname, err)
		} else {
			framework.Logf("Pod %s has the following logs: %s", podname, body)
		}
	}
}

func (t *TestStatefulsets) Create() {
	var err error
	//replicaCount := *t.statefulset.Spec.Replicas
	t.statefulset, err = t.client.AppsV1().StatefulSets(t.namespace.Name).Create(context.Background(), t.statefulset, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	// newstatefulSet.WaitForStatusReadyReplicas(t.statefulset, replicaCount)

	// pods := newstatefulSet.GetPodList(t.statefulset)
	// for _, statefulPod := range pods.Items {
	// 	t.podName = append(t.podName, statefulPod.Name)
	// }
}

func (t *TestStatefulsets) WaitForPodReady() {
	var err error
	for _, podname := range t.podName {
		err = k8sDevPod.WaitForPodCondition(t.client, t.namespace.Name, podname, failedConditionDescription, slowPodStartTimeout, podRunningCondition)
		framework.ExpectNoError(err)
	}
}

func (t *TestStatefulsets) Exec(command []string, expectedString string) {
	for _, podname := range t.podName {
		By("Statefulset Exec: executing cmd in pod")
		_, err := e2eoutput.LookForStringInPodExec(t.namespace.Name, podname, command, expectedString, execTimeout)
		framework.ExpectNoError(err)
	}
}

func (t *TestStatefulsets) drainNode() {
	By("Drain node: draining node where pod is hosted")
	cmdString := "kubectl get pod " + t.podName[0] + " -n " + t.namespace.Name + " -o jsonpath='{.spec.nodeName}'"
	fmt.Println(cmdString)
	cmd := exec.Command("bash", "-c", cmdString)
	node, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("cmd.Run() failed with %s\n", err)
		panic(err)
	}
	fmt.Printf("Node IP is:\n%s\n", string(node))

	drainString := "kubectl drain " + strings.TrimRight(string(node), "\n") + " --ignore-daemonsets --delete-local-data --force"
	fmt.Println(drainString)
	cmd = exec.Command("bash", "-c", drainString)
	out, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("Error while draining node")
		panic(err)
	}
	fmt.Println(string(out))
	// check the status of drained node
	statusCmd := "kubectl get nodes " + strings.TrimRight(string(node), "\n") + " -o jsonpath='{.spec.unschedulable}'"
	fmt.Println(statusCmd)
	cmd = exec.Command("bash", "-c", statusCmd)
	status, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("cmd.Run() failed with %s\n", err)
		panic(err)
	}
	fmt.Printf("IsNodeDrained? true/false: \n%s\n", string(status))
	if strings.TrimRight(string(status), "\n") == "true" {
		fmt.Println("Node cordoned and drained successfully")
	} else {
		fmt.Println("Node not cordoned and drained")
	}

}

func (t *TestStatefulsets) uncordonNode() {
	By("Uncordoning the drained node")
	findNode := "kubectl get nodes| grep Ready,SchedulingDisabled | awk '{print $1}'"
	cmd := exec.Command("bash", "-c", findNode)
	node, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("unable to find the cordoned node")
		fmt.Println("cmd.Run() failed with %s\n", err)
		panic(err)
	} else {
		fmt.Println("Node to uncordon %s", string(node))
	}
	uncordonCmd := "kubectl uncordon " + strings.TrimRight(string(node), "\n")
	fmt.Println(uncordonCmd)
	cmd = exec.Command("bash", "-c", uncordonCmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("unable to uncordon the node")
		fmt.Println("cmd.Run() failed with %s\n", err)
		panic(err)
	}
}

func (t *TestStatefulsets) Cleanup() {
	By("Statefulsets Cleanup: deleting Statefulset")
	e2eoutput.DumpDebugInfo(t.client, t.namespace.Name)
	t.Logs()
	framework.Logf("deleting Statefulset %q/%q", t.namespace.Name, t.statefulset.Name)
	err := t.client.AppsV1().StatefulSets(t.namespace.Name).Delete(context.Background(), t.statefulset.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}
func (s *TestSecret) Create() {
	var err error
	By("creating Secret")
	framework.Logf("creating Secret %q under %q", s.name, s.namespace)
	secret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.namespace,
		},
		StringData: map[string]string{
			"iops":      s.iops,
			"tags":      s.tags,
			"encrypted": s.encrypt,
			//"resourceGroup": resgrpID, //TODO
		},
		Data: map[string][]byte{
			"encryptionKey": []byte(s.encryptKey),
		},
		Type: v1.SecretType(s.stype),
	}
	s.secret, err = s.client.CoreV1().Secrets(s.namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
	framework.ExpectNoError(err)
}

func (s *TestSecret) Cleanup() {
	By("deleting Secret")
	framework.Logf("deleting Secret [%s]", s.secret.Name)
	err := s.client.CoreV1().Secrets(s.namespace).Delete(context.Background(), s.secret.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

type TestStorageClass struct {
	client       clientset.Interface
	storageClass *storagev1.StorageClass
	namespace    *v1.Namespace
}

type TestVolumeSnapshotClass struct {
	client              restclientset.Interface
	volumeSnapshotClass *volumesnapshotv1.VolumeSnapshotClass
	namespace           *v1.Namespace
}

func NewTestStorageClass(c clientset.Interface, ns *v1.Namespace, sc *storagev1.StorageClass) *TestStorageClass {
	return &TestStorageClass{
		client:       c,
		storageClass: sc,
		namespace:    ns,
	}
}

func (t *TestStorageClass) Create() storagev1.StorageClass {
	var err error
	By("creating StorageClass")
	framework.Logf("creating StorageClass [%s]", t.storageClass.Name)
	t.storageClass, err = t.client.StorageV1().StorageClasses().Create(context.Background(), t.storageClass, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	return *t.storageClass
}

func (t *TestStorageClass) Cleanup() {
	By("deleting a StorageClass")
	framework.Logf("deleting StorageClass [%s]", t.storageClass.Name)
	err := t.client.StorageV1().StorageClasses().Delete(context.Background(), t.storageClass.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

type TestPreProvisionedPersistentVolume struct {
	client                    clientset.Interface
	persistentVolume          *v1.PersistentVolume
	requestedPersistentVolume *v1.PersistentVolume
}

func NewTestPreProvisionedPersistentVolume(c clientset.Interface, pv *v1.PersistentVolume) *TestPreProvisionedPersistentVolume {
	return &TestPreProvisionedPersistentVolume{
		client:                    c,
		requestedPersistentVolume: pv,
	}
}

func (pv *TestPreProvisionedPersistentVolume) Create() v1.PersistentVolume {
	var err error
	By("creating a PV")
	pv.persistentVolume, err = pv.client.CoreV1().PersistentVolumes().Create(context.Background(), pv.requestedPersistentVolume, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	return *pv.persistentVolume
}

type TestPersistentVolumeClaim struct {
	name                           string
	client                         clientset.Interface
	claimSize                      string
	accessMode                     v1.PersistentVolumeAccessMode
	volumeMode                     v1.PersistentVolumeMode
	storageClass                   *storagev1.StorageClass
	namespace                      *v1.Namespace
	persistentVolume               *v1.PersistentVolume
	persistentVolumeClaim          *v1.PersistentVolumeClaim
	requestedPersistentVolumeClaim *v1.PersistentVolumeClaim
	dataSource                     *v1.TypedLocalObjectReference
}

func NewTestPersistentVolumeClaim(
	c clientset.Interface, pvcName string, ns *v1.Namespace,
	claimSize string, accessmode *v1.PersistentVolumeAccessMode,
	volumeMode VolumeMode, sc *storagev1.StorageClass) *TestPersistentVolumeClaim {

	mode := v1.PersistentVolumeFilesystem

	pvcAccessMode := v1.ReadWriteMany
	if accessmode != nil {
		pvcAccessMode = *accessmode
	}

	return &TestPersistentVolumeClaim{
		name:         pvcName,
		client:       c,
		claimSize:    claimSize,
		accessMode:   pvcAccessMode,
		volumeMode:   mode,
		namespace:    ns,
		storageClass: sc,
	}
}
func NewTestPersistentVolumeClaimWithDataSource(
	c clientset.Interface, pvcName string, ns *v1.Namespace,
	claimSize string, volumeMode VolumeMode, sc *storagev1.StorageClass, dataSource *v1.TypedLocalObjectReference) *TestPersistentVolumeClaim {
	mode := v1.PersistentVolumeFilesystem

	By("Create tpvc with data source")
	return &TestPersistentVolumeClaim{
		name:         pvcName,
		client:       c,
		claimSize:    claimSize,
		accessMode:   v1.ReadWriteMany,
		volumeMode:   mode,
		namespace:    ns,
		storageClass: sc,
		dataSource:   dataSource,
	}
}

func (t *TestPersistentVolumeClaim) Create() {
	var err error

	By("creating a PVC")
	storageClassName := ""
	if t.storageClass != nil {
		storageClassName = t.storageClass.Name
	}
	_, err = t.client.StorageV1().StorageClasses().Get(context.Background(), storageClassName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	t.requestedPersistentVolumeClaim = generatePVC(t.name, t.namespace.Name, storageClassName, t.claimSize, t.accessMode, t.volumeMode, t.dataSource)
	t.persistentVolumeClaim, err = t.client.CoreV1().PersistentVolumeClaims(t.namespace.Name).Create(context.Background(), t.requestedPersistentVolumeClaim, metav1.CreateOptions{})
	framework.ExpectNoError(err)
}

func (t *TestPersistentVolumeClaim) ValidateProvisionedPersistentVolume() {
	var err error

	// Get the bound PersistentVolume
	By("validating provisioned PV")
	t.persistentVolume, err = t.client.CoreV1().PersistentVolumes().Get(context.Background(), t.persistentVolumeClaim.Spec.VolumeName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	framework.Logf("validating provisioned PV [%s] for PVC [%s]", t.persistentVolume.Name, t.persistentVolumeClaim.Name)

	// Check sizes
	expectedCapacity := t.requestedPersistentVolumeClaim.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	claimCapacity := t.persistentVolumeClaim.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	Expect(claimCapacity.Value()).To(Equal(expectedCapacity.Value()), "claimCapacity is not equal to requestedCapacity")

	pvCapacity := t.persistentVolume.Spec.Capacity[v1.ResourceName(v1.ResourceStorage)]
	Expect(pvCapacity.Value()).To(Equal(expectedCapacity.Value()), "pvCapacity is not equal to requestedCapacity")

	// Check PV properties
	By("checking PV")
	framework.Logf("checking PV [%s]", t.persistentVolume.Name)
	expectedAccessModes := t.requestedPersistentVolumeClaim.Spec.AccessModes
	Expect(t.persistentVolume.Spec.AccessModes).To(Equal(expectedAccessModes))
	Expect(t.persistentVolume.Spec.ClaimRef.Name).To(Equal(t.persistentVolumeClaim.ObjectMeta.Name))
	Expect(t.persistentVolume.Spec.ClaimRef.Namespace).To(Equal(t.persistentVolumeClaim.ObjectMeta.Namespace))
	// If storageClass is nil, PV was pre-provisioned with these values already set
	if t.storageClass != nil {
		Expect(t.persistentVolume.Spec.PersistentVolumeReclaimPolicy).To(Equal(*t.storageClass.ReclaimPolicy))
		//Expect(t.persistentVolume.Spec.MountOptions).To(Equal(t.storageClass.MountOptions))
	}
}

func (t *TestPersistentVolumeClaim) WaitForBound() v1.PersistentVolumeClaim {
	var err error

	By(fmt.Sprintf("waiting for PVC to be in phase %q", v1.ClaimBound))
	err = k8sDevPV.WaitForPersistentVolumeClaimPhase(v1.ClaimBound, t.client, t.namespace.Name, t.persistentVolumeClaim.Name, framework.Poll, framework.ClaimProvisionTimeout)
	framework.ExpectNoError(err)

	By("checking the PVC")
	// Get new copy of the claim
	t.persistentVolumeClaim, err = t.client.CoreV1().PersistentVolumeClaims(t.namespace.Name).Get(context.Background(), t.persistentVolumeClaim.Name, metav1.GetOptions{})
	framework.ExpectNoError(err)

	return *t.persistentVolumeClaim
}

func (t *TestPersistentVolumeClaim) WaitForPending() v1.PersistentVolumeClaim {
	var err error

	By(fmt.Sprintf("waiting for PVC to be in phase %q", v1.ClaimPending))
	time.Sleep(5 * time.Minute)
	err = k8sDevPV.WaitForPersistentVolumeClaimPhase(v1.ClaimPending, t.client, t.namespace.Name, t.persistentVolumeClaim.Name, framework.Poll, framework.ClaimProvisionTimeout)
	framework.ExpectNoError(err)

	By("checking the PVC")
	// Get new copy of the claim
	t.persistentVolumeClaim, err = t.client.CoreV1().PersistentVolumeClaims(t.namespace.Name).Get(context.Background(), t.persistentVolumeClaim.Name, metav1.GetOptions{})
	framework.ExpectNoError(err)

	return *t.persistentVolumeClaim
}

func generatePVC(name, namespace,
	storageClassName, claimSize string,
	accessMode v1.PersistentVolumeAccessMode,
	volumeMode v1.PersistentVolumeMode, dataSource *v1.TypedLocalObjectReference) *v1.PersistentVolumeClaim {
	var objMeta metav1.ObjectMeta
	lastChar := name[len(name)-1:]
	if lastChar == "-" {
		objMeta = metav1.ObjectMeta{
			GenerateName: name,
			Namespace:    namespace,
			Labels:       map[string]string{"app": "ics-e2e-tester"},
		}
	} else {
		objMeta = metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": "ics-e2e-tester"},
		}
	}
	return &v1.PersistentVolumeClaim{
		ObjectMeta: objMeta,
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []v1.PersistentVolumeAccessMode{
				accessMode,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(claimSize),
				},
			},
			VolumeMode: &volumeMode,
			DataSource: dataSource,
		},
	}
}

func (t *TestPersistentVolumeClaim) Cleanup() {

	volumeHandle := strings.Split(t.persistentVolume.Spec.PersistentVolumeSource.CSI.VolumeHandle, ":")
	getShareMountTargetOptions := &vpcbetav1.GetShareMountTargetOptions{
		ShareID: &volumeHandle[0],
		ID:      &volumeHandle[1],
	}

	By(fmt.Sprintf("Logging volumeHandle", volumeHandle))
	By(fmt.Sprintf("Verifying if file share mount target is created"))
	shareMountTarget, response, err := VPCService.GetShareMountTarget(getShareMountTargetOptions)
	if err != nil {
		panic(err)
	}
	// end-get_share_mount_target

	Expect(err).To(BeNil())
	Expect(response.StatusCode).To(Equal(200))
	Expect(shareMountTarget).ToNot(BeNil())

	By(fmt.Sprintf("Verifying if VNI is created"))
	if *(shareMountTarget.AccessControlMode) == "security_group" {

		getVirtualNetworkInterfaceOptions := &vpcbetav1.GetVirtualNetworkInterfaceOptions{
			ID: shareMountTarget.VirtualNetworkInterface.ID,
		}

		virtualNetworkInterface, response, err := VPCService.GetVirtualNetworkInterface(getVirtualNetworkInterfaceOptions)
		if err != nil {
			panic(err)
		}

		// end-get_virtual_network_interface

		Expect(err).To(BeNil())
		Expect(response.StatusCode).To(Equal(200))
		Expect(virtualNetworkInterface).ToNot(BeNil())
	}

	By(fmt.Sprintf("deleting PVC [%s]", t.persistentVolumeClaim.Name))
	err = k8sDevPV.DeletePersistentVolumeClaim(t.client, t.persistentVolumeClaim.Name, t.namespace.Name)
	By("Triggered DeletePersistentVolumeClaim call")
	framework.ExpectNoError(err)
	// Wait for the PV to get deleted if reclaim policy is Delete. (If it's
	// Retain, there's no use waiting because the PV won't be auto-deleted and
	// it's expected for the caller to do it.) Technically, the first few delete
	// attempts may fail, as the volume is still attached to a node because
	// kubelet is slowly cleaning up the previous pod, however it should succeed
	// in a couple of minutes.
	if t.persistentVolume != nil && t.persistentVolume.Spec.PersistentVolumeReclaimPolicy == v1.PersistentVolumeReclaimDelete {
		framework.Logf("deleting PVC [%s/%s] using PV [%s]", t.namespace.Name, t.persistentVolumeClaim.Name, t.persistentVolume.Name)
		By(fmt.Sprintf("waiting for claim's PV [%s] to be deleted", t.persistentVolume.Name))
		err := k8sDevPV.WaitForPersistentVolumeDeleted(t.client, t.persistentVolume.Name, 5*time.Second, 10*time.Minute)
		framework.ExpectNoError(err)
	}

	//Now PVC is deleted wait for some time and display that PV is still exists
	if t.persistentVolume != nil && t.persistentVolume.Spec.PersistentVolumeReclaimPolicy == v1.PersistentVolumeReclaimRetain {
		err = k8sDevPV.WaitForPersistentVolumeDeleted(t.client, t.persistentVolume.Name, 5*time.Second, 2*time.Minute)
		framework.ExpectError(err)
		By(fmt.Sprintf("Deleting PV object [%s]", t.persistentVolume.Name))
		err = k8sDevPV.DeletePersistentVolume(t.client, t.persistentVolume.Name)
		framework.ExpectNoError(err)

		//Manually delete file share target and file share as k8s on deletion of PV object does not invoke CSI Driver controlleDeleteVolume
		deleteShareMountTargetOptions := &vpcbetav1.DeleteShareMountTargetOptions{
			ShareID: &volumeHandle[0],
			ID:      &volumeHandle[1],
		}

		By(fmt.Sprintf("Deleting file share target"))
		shareMountTarget, response, err := VPCService.DeleteShareMountTarget(deleteShareMountTargetOptions)
		if err != nil {
			panic(err)
		}

		By(fmt.Sprintf("Wating for file share target to be deleted"))
		// end-delete_share_mount_target
		time.Sleep(1 * time.Minute)

		Expect(err).To(BeNil())
		Expect(response.StatusCode).To(Equal(202))
		Expect(shareMountTarget).ToNot(BeNil())

		deleteShareOptions := &vpcbetav1.DeleteShareOptions{
			ID: &volumeHandle[0],
		}

		By(fmt.Sprintf("Deleting file share"))
		share, response, err := VPCService.DeleteShare(deleteShareOptions)
		if err != nil {
			panic(err)
		}

		// end-delete_share

		Expect(err).To(BeNil())
		Expect(response.StatusCode).To(Equal(202))
		Expect(share).ToNot(BeNil())
	}

	By(fmt.Sprintf("Wating for file share to be deleted"))
	// end-delete_share_mount_target
	time.Sleep(30 * time.Second)

	By(fmt.Sprintf("Verying if VNI is deleted"))
	//Verifying if VNI is deleted
	if *(shareMountTarget.AccessControlMode) == "security_group" {

		getVirtualNetworkInterfaceOptions := &vpcbetav1.GetVirtualNetworkInterfaceOptions{
			ID: shareMountTarget.VirtualNetworkInterface.ID,
		}

		virtualNetworkInterface, response, err := VPCService.GetVirtualNetworkInterface(getVirtualNetworkInterfaceOptions)

		// end-get_virtual_network_interface
		Expect(err).ToNot(BeNil())
		Expect(response.StatusCode).To(Equal(404))
		Expect(virtualNetworkInterface).To(BeNil())
	}

	By(fmt.Sprintf("Verying if shareMountTarget is deleted"))
	//Verifying if shareMountTarget is deleted
	shareMountTarget, response, err = VPCService.GetShareMountTarget(getShareMountTargetOptions)
	// end-get_share_mount_target

	Expect(err).ToNot(BeNil())
	Expect(response.StatusCode).To(Equal(404))
	Expect(shareMountTarget).To(BeNil())

	By(fmt.Sprintf("Verying if share is deleted"))
	//Verifying if share is deleted
	getShareOptions := &vpcbetav1.GetShareOptions{
		ID: &volumeHandle[1],
	}
	share, response, err := VPCService.GetShare(getShareOptions)
	// end-get_share

	Expect(err).ToNot(BeNil())
	Expect(response.StatusCode).To(Equal(404))
	Expect(share).To(BeNil())
}

func (t *TestPersistentVolumeClaim) ReclaimPolicy() v1.PersistentVolumeReclaimPolicy {
	return t.persistentVolume.Spec.PersistentVolumeReclaimPolicy
}

func (t *TestPersistentVolumeClaim) WaitForPersistentVolumePhase(phase v1.PersistentVolumePhase) {
	err := k8sDevPV.WaitForPersistentVolumePhase(phase, t.client, t.persistentVolume.Name, 5*time.Second, 10*time.Minute)
	framework.ExpectNoError(err)
}

func (t *TestPersistentVolumeClaim) DeleteBoundPersistentVolume() {
	By("deleting PV")
	framework.Logf("deleting PV [%s]", t.persistentVolume.Name)
	err := k8sDevPV.DeletePersistentVolume(t.client, t.persistentVolume.Name)
	framework.ExpectNoError(err)
	By(fmt.Sprintf("waiting for claim's PV %q to be deleted", t.persistentVolume.Name))
	err = k8sDevPV.WaitForPersistentVolumeDeleted(t.client, t.persistentVolume.Name, 5*time.Second, 10*time.Minute)
	framework.ExpectNoError(err)
}

type TestDeployment struct {
	client     clientset.Interface
	deployment *apps.Deployment
	namespace  *v1.Namespace
	podName    string
}

func NewTestDeployment(c clientset.Interface, ns *v1.Namespace, command string, pvc *v1.PersistentVolumeClaim, volumeName, mountPath string, readOnly bool, replicaCount int32) *TestDeployment {
	generateName := "ics-e2e-tester-"
	selectorValue := fmt.Sprintf("%s%d", generateName, rand.Int())
	return &TestDeployment{
		client:    c,
		namespace: ns,
		deployment: &apps.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
			},
			Spec: apps.DeploymentSpec{
				Replicas: &replicaCount,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": selectorValue},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": selectorValue},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:    "ics-e2e-tester",
								Image:   imageutils.GetE2EImage(imageutils.BusyBox),
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", command},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      volumeName,
										MountPath: mountPath,
										ReadOnly:  readOnly,
									},
								},
							},
						},
						RestartPolicy: v1.RestartPolicyAlways,
						Volumes: []v1.Volume{
							{
								Name: volumeName,
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: pvc.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewTestDeploymentWitoutPVC(c clientset.Interface, ns *v1.Namespace, command string) *TestDeployment {
	generateName := "ics-e2e-tester-"
	selectorValue := fmt.Sprintf("%s%d", generateName, rand.Int())
	replicas := int32(1)
	return &TestDeployment{
		client:    c,
		namespace: ns,
		deployment: &apps.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
			},
			Spec: apps.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": selectorValue},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": selectorValue},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:         "ics-e2e-tester",
								Image:        imageutils.GetE2EImage(imageutils.BusyBox),
								Command:      []string{"/bin/sh"},
								Args:         []string{"-c", command},
								VolumeMounts: make([]v1.VolumeMount, 0),
							},
						},
						RestartPolicy: v1.RestartPolicyAlways,
						Volumes:       make([]v1.Volume, 0),
					},
				},
			},
		},
	}
}

func (t *TestDeployment) SetupVolume(pvc *v1.PersistentVolumeClaim, name, mountPath string, readOnly bool) {
	volumeMount := v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
	t.deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(t.deployment.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)

	volume := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.Name,
			},
		},
	}
	t.deployment.Spec.Template.Spec.Volumes = append(t.deployment.Spec.Template.Spec.Volumes, volume)
}

func (t *TestDeployment) Create() {
	var err error
	t.deployment, err = t.client.AppsV1().Deployments(t.namespace.Name).Create(context.Background(), t.deployment, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	err = k8sDevDep.WaitForDeploymentComplete(t.client, t.deployment)
	framework.ExpectNoError(err)

	pods, err := k8sDevDep.GetPodsForDeployment(t.client, t.deployment)
	framework.ExpectNoError(err)
	// always get first pod as there should only be one
	t.podName = pods.Items[0].Name

	//POD Creation takes time becuase of volume attac/detach
	//err = framework.WaitForPodSuccessInNamespaceSlow(t.client, t.podName, t.namespace.Name)
	//framework.ExpectNoError(err)
	//err = framework.WaitForDeploymentComplete(t.client, t.deployment)
	//framework.ExpectNoError(err)
}

func (t *TestDeployment) WaitForPodReady() {
	pods, err := k8sDevDep.GetPodsForDeployment(t.client, t.deployment)
	framework.ExpectNoError(err)
	// always get first pod as there should only be one
	pod := pods.Items[0]
	t.podName = pod.Name
	err = k8sDevPod.WaitForPodRunningInNamespace(t.client, &pod)
	framework.ExpectNoError(err)
}

func (t *TestDeployment) Exec(command []string, expectedString string) {
	By("Deployment Exec: executing cmd in POD")
	framework.Logf("executing cmd in POD [%s/%s]", t.namespace.Name, t.podName)
	_, err := e2eoutput.LookForStringInPodExec(t.namespace.Name, t.podName, command, expectedString, execTimeout)
	framework.ExpectNoError(err)
}

func (t *TestDeployment) DeletePodAndWait() {
	By("Deployment DeletePodAndWait: deleting POD")
	framework.Logf("deleting POD [%s/%s]", t.namespace.Name, t.podName)
	e2eoutput.DumpDebugInfo(t.client, t.namespace.Name)
	err := t.client.CoreV1().Pods(t.namespace.Name).Delete(context.Background(), t.podName, metav1.DeleteOptions{})
	if err != nil {
		if !apierrs.IsNotFound(err) {
			framework.ExpectNoError(fmt.Errorf("pod %q Delete API error: %v", t.podName, err))
		}
		return
	}
	framework.Logf("Waiting for pod [%s/%s] to be fully deleted", t.namespace.Name, t.podName)
	err = k8sDevPod.WaitForPodNotFoundInNamespace(t.client, t.podName, t.namespace.Name, 60*time.Second)
	if err != nil {
		framework.ExpectNoError(fmt.Errorf("pod [%s] error waiting for delete: %v", t.podName, err))
	}
}

func (t *TestDeployment) Cleanup() {
	By("Deployment Cleanup: deleting Deployment")
	framework.Logf("deleting Deployment [%s/%s]", t.namespace.Name, t.deployment.Name)
	e2eoutput.DumpDebugInfo(t.client, t.namespace.Name)
	t.Logs()
	err := t.client.AppsV1().Deployments(t.namespace.Name).Delete(context.Background(), t.deployment.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (t *TestDeployment) Logs() {
	body, err := podLogs(t.client, t.podName, t.namespace.Name)
	if err != nil {
		framework.Logf("Error getting logs for pod [%s]: %v", t.podName, err)
	} else {
		framework.Logf("Pod [%s] has the following logs: %s", t.podName, body)
	}
}

type TestPod struct {
	client      clientset.Interface
	pod         *v1.Pod
	namespace   *v1.Namespace
	dumpDbgInfo bool
	dumpLog     bool
}

func NewTestPodWithName(c clientset.Interface, ns *v1.Namespace, name, command string) *TestPod {
	return &TestPod{
		dumpDbgInfo: true,
		dumpLog:     true,
		client:      c,
		namespace:   ns,
		pod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: name,
				Labels:       map[string]string{"app": "ics-vol-e2e"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:         "ics-e2e-tester",
						Image:        imageutils.GetE2EImage(imageutils.BusyBox),
						Command:      []string{"/bin/sh"},
						Args:         []string{"-c", command},
						VolumeMounts: make([]v1.VolumeMount, 0),
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
				Volumes:       make([]v1.Volume, 0),
			},
		},
	}
}

func NewTestPod(c clientset.Interface, ns *v1.Namespace, command string) *TestPod {
	return &TestPod{
		dumpDbgInfo: true,
		dumpLog:     true,
		client:      c,
		namespace:   ns,
		pod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "ics-e2e-tester-",
				Labels:       map[string]string{"app": "ics-vol-e2e"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:         "ics-e2e-tester",
						Image:        imageutils.GetE2EImage(imageutils.BusyBox),
						Command:      []string{"/bin/sh"},
						Args:         []string{"-c", command},
						VolumeMounts: make([]v1.VolumeMount, 0),
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
				Volumes:       make([]v1.Volume, 0),
			},
		},
	}
}

func (t *TestPod) Create() {
	var err error

	t.pod, err = t.client.CoreV1().Pods(t.namespace.Name).Create(context.Background(), t.pod, metav1.CreateOptions{})
	framework.ExpectNoError(err)
}

func (t *TestPod) Delete() {
	By("POD Delete: deleting POD")
	framework.Logf("deleting POD [%s/%s]", t.namespace.Name, t.pod.Name)
	e2eoutput.DumpDebugInfo(t.client, t.namespace.Name)
	err := t.client.CoreV1().Pods(t.namespace.Name).Delete(context.Background(), t.pod.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (t *TestPod) Exec(command []string, expectedString string) {
	By("POD Exec: executing cmd in POD")
	framework.Logf("executing cmd in POD [%s/%s]", t.namespace.Name, t.pod.Name)
	_, err := e2eoutput.LookForStringInPodExec(t.namespace.Name, t.pod.Name, command, expectedString, execTimeout)
	framework.ExpectNoError(err)
}

func (t *TestPod) WaitForSuccess() {
	By(fmt.Sprintf("checking that the pods command exits with no error [%s/%s]", t.namespace.Name, t.pod.Name))
	err := k8sDevPod.WaitForPodSuccessInNamespaceSlow(t.client, t.pod.Name, t.namespace.Name)
	framework.ExpectNoError(err)
}

var podRunningCondition = func(pod *v1.Pod) (bool, error) {
	switch pod.Status.Phase {
	case v1.PodRunning:
		By("Saw pod running")
		return true, nil
	case v1.PodSucceeded:
		return true, fmt.Errorf("pod %q successed with reason: %q, message: %q", pod.Name, pod.Status.Reason, pod.Status.Message)
	default:
		return false, nil
	}
}

func (t *TestPod) WaitForRunningSlow() {
	By(fmt.Sprintf("checking that the pods status is running [%s/%s]", t.namespace.Name, t.pod.Name))
	//err := framework.WaitTimeoutForPodRunningInNamespace(t.client, t.pod.Name, t.namespace.Name, slowPodStartTimeout)
	err := k8sDevPod.WaitForPodCondition(t.client, t.namespace.Name, t.pod.Name, failedConditionDescription, slowPodStartTimeout, podRunningCondition)
	framework.ExpectNoError(err)
}

func (t *TestPod) WaitForRunning() {
	err := k8sDevPod.WaitForPodRunningInNamespace(t.client, t.pod)
	framework.ExpectNoError(err)
}

var podFailedCondition = func(pod *v1.Pod) (bool, error) {
	switch pod.Status.Phase {
	case v1.PodFailed:
		By("Saw pod failure")
		return true, nil
	case v1.PodSucceeded:
		return true, fmt.Errorf("pod %q successed with reason: %q, message: %q", pod.Name, pod.Status.Reason, pod.Status.Message)
	default:
		return false, nil
	}
}

func (t *TestPod) WaitForFailure() {
	err := k8sDevPod.WaitForPodCondition(t.client, t.namespace.Name, t.pod.Name, failedConditionDescription, slowPodStartTimeout, podFailedCondition)
	framework.ExpectNoError(err)
}

func (t *TestPod) SetupVolume(pvc *v1.PersistentVolumeClaim, name, mountPath string, readOnly bool) {
	volumeMount := v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
	t.pod.Spec.Containers[0].VolumeMounts = append(t.pod.Spec.Containers[0].VolumeMounts, volumeMount)

	volume := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.Name,
			},
		},
	}
	t.pod.Spec.Volumes = append(t.pod.Spec.Volumes, volume)
}

func (t *TestPod) SetNodeSelector(nodeSelector map[string]string) {
	t.pod.Spec.NodeSelector = nodeSelector
}

func (t *TestPod) Cleanup() {
	By("POD Cleanup: deleting POD")
	framework.Logf("deleting POD [%s/%s]", t.namespace.Name, t.pod.Name)
	cleanupPodOrFail(t.client, t.pod.Name, t.namespace.Name, t.dumpDbgInfo, t.dumpLog)
}

func (t *TestPod) DumpLogOff() {
	t.dumpLog = false
}

func (t *TestPod) DumpDbgInfoOff() {
	t.dumpDbgInfo = false
}

func (t *TestPod) Logs() {
	body, err := podLogs(t.client, t.pod.Name, t.namespace.Name)
	if err != nil {
		framework.Logf("Error getting logs for pod %s: %v", t.pod.Name, err)
	} else {
		framework.Logf("Pod %s has the following logs: %s", t.pod.Name, body)
	}
}

func cleanupPodOrFail(client clientset.Interface, name, namespace string, dbginfo, log bool) {
	if dbginfo {
		e2eoutput.DumpDebugInfo(client, namespace)
	}
	if log {
		body, err := podLogs(client, name, namespace)
		if err != nil {
			framework.Logf("Error getting logs for pod %s: %v", name, err)
		} else {
			framework.Logf("Pod %s has the following logs: %s", name, body)
		}
	}
	framework.Logf("deleting POD [%s/%s]", namespace, name)
	k8sDevPod.DeletePodOrFail(client, namespace, name)
}

func podLogs(client clientset.Interface, name, namespace string) ([]byte, error) {
	return client.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{}).Do(context.Background()).Raw()
}

func NewTestVolumeSnapshotClass(c restclientset.Interface, ns *v1.Namespace, vsc *volumesnapshotv1.VolumeSnapshotClass) *TestVolumeSnapshotClass {
	return &TestVolumeSnapshotClass{
		client:              c,
		volumeSnapshotClass: vsc,
		namespace:           ns,
	}
}

func (t *TestVolumeSnapshotClass) Create() {
	By("creating a VolumeSnapshotClass")
	var err error
	t.volumeSnapshotClass, err = snapshotclientset.New(t.client).SnapshotV1().VolumeSnapshotClasses().Create(context.TODO(), t.volumeSnapshotClass, metav1.CreateOptions{})
	framework.ExpectNoError(err)
}

func (t *TestVolumeSnapshotClass) CreateSnapshot(pvc *v1.PersistentVolumeClaim) *volumesnapshotv1.VolumeSnapshot {
	By("creating a VolumeSnapshot for " + pvc.Name)
	snapshot := &volumesnapshotv1.VolumeSnapshot{
		TypeMeta: metav1.TypeMeta{
			Kind:       VolumeSnapshotKind,
			APIVersion: SnapshotAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "volume-snapshot-",
			Namespace:    t.namespace.Name,
		},
		Spec: volumesnapshotv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &t.volumeSnapshotClass.Name,
			Source: volumesnapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvc.Name,
			},
		},
	}
	snapshot, err := snapshotclientset.New(t.client).SnapshotV1().VolumeSnapshots(t.namespace.Name).Create(context.TODO(), snapshot, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	return snapshot
}

func (t *TestVolumeSnapshotClass) ReadyToUse(snapshot *volumesnapshotv1.VolumeSnapshot, snapFail bool) {
	By("waiting for VolumeSnapshot to be ready to use - " + snapshot.Name)
	err := wait.Poll(15*time.Second, 5*time.Minute, func() (bool, error) {
		vs, err := snapshotclientset.New(t.client).SnapshotV1().VolumeSnapshots(t.namespace.Name).Get(context.TODO(), snapshot.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("did not see ReadyToUse: %v", err)
		}

		if vs.Status == nil || vs.Status.ReadyToUse == nil {
			return false, nil
		}
		return *vs.Status.ReadyToUse, nil
	})
	if snapFail == true {
		framework.ExpectError(err)
	} else {
		framework.ExpectNoError(err)
	}
}

func (t *TestVolumeSnapshotClass) DeleteSnapshot(vs *volumesnapshotv1.VolumeSnapshot) {
	By("deleting a VolumeSnapshot " + vs.Name)
	err := snapshotclientset.New(t.client).SnapshotV1().VolumeSnapshots(t.namespace.Name).Delete(context.TODO(), vs.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)

	err = t.waitForSnapshotDeleted(t.namespace.Name, vs.Name, 2*time.Second, 15*time.Minute)
	framework.ExpectNoError(err)
}

func (t *TestVolumeSnapshotClass) Cleanup() {
	framework.Logf("deleting VolumeSnapshotClass %s", t.volumeSnapshotClass.Name)
	err := snapshotclientset.New(t.client).SnapshotV1().VolumeSnapshotClasses().Delete(context.TODO(), t.volumeSnapshotClass.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (t *TestVolumeSnapshotClass) waitForSnapshotDeleted(ns string, snapshotName string, poll, timeout time.Duration) error {
	framework.Logf("Waiting up to %v for VolumeSnapshot %s to be removed", timeout, snapshotName)
	c := snapshotclientset.New(t.client).SnapshotV1()
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		_, err := c.VolumeSnapshots(ns).Get(context.TODO(), snapshotName, metav1.GetOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				framework.Logf("Snapshot %q in namespace %q doesn't exist in the system", snapshotName, ns)
				return nil
			}
			framework.Logf("Failed to get snapshot %q in namespace %q, retrying in %v. Error: %v", snapshotName, ns, poll, err)
		}
	}
	return fmt.Errorf("VolumeSnapshot %s is not removed from the system within %v", snapshotName, timeout)
}

func GetVolumeSnapshotClass(namespace string) *volumesnapshotv1.VolumeSnapshotClass {
	provisioner := "vpc.file.csi.ibm.io"
	generateName := fmt.Sprintf("%s-%s-dynamic-sc-", namespace, provisioner)
	return getVolumeSnapshotClass(generateName, provisioner)
}

func getVolumeSnapshotClass(generateName string, provisioner string) *volumesnapshotv1.VolumeSnapshotClass {
	return &volumesnapshotv1.VolumeSnapshotClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       VolumeSnapshotClassKind,
			APIVersion: SnapshotAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName,
		},
		Driver:         provisioner,
		DeletionPolicy: volumesnapshotv1.VolumeSnapshotContentDelete,
	}
}
