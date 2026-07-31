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

package testsuites

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	volumeGroupSnapshotAPIGroup = "groupsnapshot.storage.k8s.io"
	volumeGroupSnapshotDriver   = "vpc.block.csi.ibm.io"
	volumeGroupSnapshotLabelKey = "vgs-e2e.ibm.com/group"
	volumeGroupSnapshotPoll     = 15 * time.Second
	volumeGroupSnapshotTimeout  = 15 * time.Minute
)

var supportedVolumeGroupSnapshotVersions = []string{"v1beta2", "v1beta1"}

// DynamicallyProvisionedVolumeGroupSnapshotTest validates creation, restore,
// data integrity, and deletion of a dynamically provisioned volume group snapshot.
type DynamicallyProvisionedVolumeGroupSnapshotTest struct {
	Volumes []VolumeDetails
}

type volumeGroupSnapshotResources struct {
	apiVersion           string
	groupSnapshot        schema.GroupVersionResource
	groupSnapshotClass   schema.GroupVersionResource
	groupSnapshotContent schema.GroupVersionResource
	volumeSnapshot       schema.GroupVersionResource
}

type volumeGroupSnapshotMember struct {
	volumeHandle string
	snapshotName string
}

// Run executes the volume group snapshot lifecycle for all configured volumes.
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) Run(client clientset.Interface, dynamicClient dynamic.Interface, namespace *v1.Namespace) {
	Expect(t.Volumes).To(HaveLen(2), "the VGS E2E test requires exactly two source volumes")

	resources := discoverVolumeGroupSnapshotResources(client)
	groupLabelValue := namespace.Name
	sourcePVCs := make([]*TestPersistentVolumeClaim, 0, len(t.Volumes))
	volumeHandleToExpectedData := make(map[string]string, len(t.Volumes))
	volumeHandles := make([]string, 0, len(t.Volumes))

	By("provisioning and labeling source PVCs for the volume group snapshot")
	for index := range t.Volumes {
		volume := &t.Volumes[index]
		tpvc, cleanupFuncs := volume.SetupDynamicPersistentVolumeClaim(client, namespace, false)
		for _, cleanup := range cleanupFuncs {
			defer cleanup()
		}

		pvc := tpvc.persistentVolumeClaim.DeepCopy()
		if pvc.Labels == nil {
			pvc.Labels = make(map[string]string)
		}
		pvc.Labels[volumeGroupSnapshotLabelKey] = groupLabelValue
		updatedPVC, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Update(context.Background(), pvc, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
		tpvc.persistentVolumeClaim = updatedPVC

		if tpvc.persistentVolume == nil || tpvc.persistentVolume.Spec.CSI == nil {
			Fail(fmt.Sprintf("PVC %s is not backed by a CSI persistent volume", updatedPVC.Name))
		}
		volumeHandle := tpvc.persistentVolume.Spec.CSI.VolumeHandle
		expectedData := fmt.Sprintf("vgs-volume-%d", index+1)
		volumeHandleToExpectedData[volumeHandle] = expectedData
		volumeHandles = append(volumeHandles, volumeHandle)
		sourcePVCs = append(sourcePVCs, tpvc)
	}

	By("writing distinct data to each source volume")
	writeCommands := make([]string, 0, len(sourcePVCs)+2)
	sourcePod := NewTestPod(client, namespace, "")
	for index, tpvc := range sourcePVCs {
		mountPath := fmt.Sprintf("/mnt/vgs-source-%d", index+1)
		sourcePod.SetupVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("vgs-source-%d", index+1), mountPath, false)
		writeCommands = append(writeCommands, fmt.Sprintf("echo %q > %s/data", volumeHandleToExpectedData[volumeHandles[index]], mountPath))
	}
	writeCommands = append(writeCommands, "sync", "while true; do sleep 2; done")
	sourcePod.pod.Spec.Containers[0].Args = []string{"-c", strings.Join(writeCommands, " && ")}
	sourcePod.Create()
	defer sourcePod.Cleanup()
	sourcePod.WaitForRunningSlow()
	for index, volumeHandle := range volumeHandles {
		sourcePod.Exec([]string{"cat", fmt.Sprintf("/mnt/vgs-source-%d/data", index+1)}, volumeHandleToExpectedData[volumeHandle]+"\n")
	}

	By("creating a VolumeGroupSnapshotClass")
	groupSnapshotClass := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": resources.apiVersion,
		"kind":       "VolumeGroupSnapshotClass",
		"metadata": map[string]interface{}{
			"generateName": namespace.Name + "-vgs-class-",
		},
		"driver":         volumeGroupSnapshotDriver,
		"deletionPolicy": "Delete",
	}}
	groupSnapshotClass, err := dynamicClient.Resource(resources.groupSnapshotClass).Create(context.Background(), groupSnapshotClass, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	defer func() {
		By("deleting the VolumeGroupSnapshotClass")
		err := dynamicClient.Resource(resources.groupSnapshotClass).Delete(context.Background(), groupSnapshotClass.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrs.IsNotFound(err) {
			framework.ExpectNoError(err)
		}
	}()

	By("creating a VolumeGroupSnapshot for the labeled PVCs")
	groupSnapshot := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": resources.apiVersion,
		"kind":       "VolumeGroupSnapshot",
		"metadata": map[string]interface{}{
			"generateName": "volume-group-snapshot-",
			"namespace":    namespace.Name,
		},
		"spec": map[string]interface{}{
			"volumeGroupSnapshotClassName": groupSnapshotClass.GetName(),
			"source": map[string]interface{}{
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						volumeGroupSnapshotLabelKey: groupLabelValue,
					},
				},
			},
		},
	}}
	groupSnapshot, err = dynamicClient.Resource(resources.groupSnapshot).Namespace(namespace.Name).Create(context.Background(), groupSnapshot, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	contentName := ""
	members := make([]volumeGroupSnapshotMember, 0, len(t.Volumes))
	defer func() {
		By("deleting the VolumeGroupSnapshot and its generated resources")
		err := dynamicClient.Resource(resources.groupSnapshot).Namespace(namespace.Name).Delete(context.Background(), groupSnapshot.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrs.IsNotFound(err) {
			framework.ExpectNoError(err)
		}
		framework.ExpectNoError(waitForVolumeGroupSnapshotDeletion(dynamicClient, resources, namespace.Name, groupSnapshot.GetName(), contentName, members))
	}()

	By("waiting for the VolumeGroupSnapshot to become ready")
	groupSnapshot, err = waitForVolumeGroupSnapshotReady(dynamicClient, resources.groupSnapshot, namespace.Name, groupSnapshot.GetName())
	framework.ExpectNoError(err)
	contentName, _, err = unstructured.NestedString(groupSnapshot.Object, "status", "boundVolumeGroupSnapshotContentName")
	framework.ExpectNoError(err)
	Expect(contentName).NotTo(BeEmpty(), "the VolumeGroupSnapshot must be bound to content")

	groupSnapshotContent, err := dynamicClient.Resource(resources.groupSnapshotContent).Get(context.Background(), contentName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	validateVolumeGroupSnapshotBinding(groupSnapshot, groupSnapshotContent, namespace.Name)

	By("validating all member snapshots in the group")
	members, err = getVolumeGroupSnapshotMembers(groupSnapshot, groupSnapshotContent, len(t.Volumes))
	framework.ExpectNoError(err)
	for _, member := range members {
		_, found := volumeHandleToExpectedData[member.volumeHandle]
		Expect(found).To(BeTrue(), "group snapshot contains unexpected volume handle %q", member.volumeHandle)
		framework.ExpectNoError(waitForVolumeSnapshotReady(dynamicClient, resources.volumeSnapshot, namespace.Name, member.snapshotName, groupSnapshot.GetName()))
	}

	By("restoring a PVC from each member snapshot")
	memberByVolumeHandle := make(map[string]string, len(members))
	for _, member := range members {
		memberByVolumeHandle[member.volumeHandle] = member.snapshotName
	}
	restoredPVCs := make([]*TestPersistentVolumeClaim, 0, len(t.Volumes))
	for index, volumeHandle := range volumeHandles {
		snapshotName, found := memberByVolumeHandle[volumeHandle]
		Expect(found).To(BeTrue(), "no member snapshot was generated for source volume %q", volumeHandle)
		sourceVolume := t.Volumes[index]
		restoredVolume := VolumeDetails{
			PVCName:       fmt.Sprintf("ics-vgs-restored-%d-", index+1),
			VolumeType:    sourceVolume.VolumeType,
			FSType:        sourceVolume.FSType,
			ClaimSize:     sourceVolume.ClaimSize,
			ReclaimPolicy: sourceVolume.ReclaimPolicy,
			MountOptions:  sourceVolume.MountOptions,
			AccessMode:    sourceVolume.AccessMode,
			VolumeMode:    sourceVolume.VolumeMode,
			DataSource:    &DataSource{Name: snapshotName},
		}
		tpvc, cleanupFuncs := restoredVolume.SetupDynamicPersistentVolumeClaim(client, namespace, false)
		for _, cleanup := range cleanupFuncs {
			defer cleanup()
		}
		restoredPVCs = append(restoredPVCs, tpvc)
	}

	By("verifying data on every restored volume")
	restoredPod := NewTestPod(client, namespace, "while true; do sleep 2; done")
	for index, tpvc := range restoredPVCs {
		restoredPod.SetupVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("vgs-restored-%d", index+1), fmt.Sprintf("/mnt/vgs-restored-%d", index+1), false)
	}
	restoredPod.Create()
	defer restoredPod.Cleanup()
	restoredPod.WaitForRunningSlow()
	for index, volumeHandle := range volumeHandles {
		restoredPod.Exec([]string{"cat", fmt.Sprintf("/mnt/vgs-restored-%d/data", index+1)}, volumeHandleToExpectedData[volumeHandle]+"\n")
	}
}

func discoverVolumeGroupSnapshotResources(client clientset.Interface) volumeGroupSnapshotResources {
	for _, version := range supportedVolumeGroupSnapshotVersions {
		groupVersion := volumeGroupSnapshotAPIGroup + "/" + version
		resourceList, err := client.Discovery().ServerResourcesForGroupVersion(groupVersion)
		if err != nil {
			continue
		}

		availableResources := make(map[string]bool, len(resourceList.APIResources))
		for _, resource := range resourceList.APIResources {
			availableResources[resource.Name] = true
		}
		if !availableResources["volumegroupsnapshots"] || !availableResources["volumegroupsnapshotclasses"] || !availableResources["volumegroupsnapshotcontents"] {
			continue
		}

		return volumeGroupSnapshotResources{
			apiVersion:           groupVersion,
			groupSnapshot:        schema.GroupVersionResource{Group: volumeGroupSnapshotAPIGroup, Version: version, Resource: "volumegroupsnapshots"},
			groupSnapshotClass:   schema.GroupVersionResource{Group: volumeGroupSnapshotAPIGroup, Version: version, Resource: "volumegroupsnapshotclasses"},
			groupSnapshotContent: schema.GroupVersionResource{Group: volumeGroupSnapshotAPIGroup, Version: version, Resource: "volumegroupsnapshotcontents"},
			volumeSnapshot:       schema.GroupVersionResource{Group: SnapshotAPIGroup, Version: APIVersionv1, Resource: "volumesnapshots"},
		}
	}

	Fail("the cluster does not serve the VolumeGroupSnapshot v1beta2 or v1beta1 APIs")
	return volumeGroupSnapshotResources{}
}

func waitForVolumeGroupSnapshotReady(dynamicClient dynamic.Interface, resource schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	var readySnapshot *unstructured.Unstructured
	err := waitForResourceCondition(func() (bool, error) {
		groupSnapshot, err := dynamicClient.Resource(resource).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		ready, _, err := unstructured.NestedBool(groupSnapshot.Object, "status", "readyToUse")
		if err != nil || !ready {
			return false, err
		}
		readySnapshot = groupSnapshot
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("VolumeGroupSnapshot %s/%s did not become ready: %w", namespace, name, err)
	}
	return readySnapshot, nil
}

func validateVolumeGroupSnapshotBinding(groupSnapshot, content *unstructured.Unstructured, namespace string) {
	boundName, _, err := unstructured.NestedString(groupSnapshot.Object, "status", "boundVolumeGroupSnapshotContentName")
	framework.ExpectNoError(err)
	Expect(boundName).To(Equal(content.GetName()))

	referenceName, _, err := unstructured.NestedString(content.Object, "spec", "volumeGroupSnapshotRef", "name")
	framework.ExpectNoError(err)
	referenceNamespace, _, err := unstructured.NestedString(content.Object, "spec", "volumeGroupSnapshotRef", "namespace")
	framework.ExpectNoError(err)
	driver, _, err := unstructured.NestedString(content.Object, "spec", "driver")
	framework.ExpectNoError(err)
	ready, _, err := unstructured.NestedBool(content.Object, "status", "readyToUse")
	framework.ExpectNoError(err)

	Expect(referenceName).To(Equal(groupSnapshot.GetName()))
	Expect(referenceNamespace).To(Equal(namespace))
	Expect(driver).To(Equal(volumeGroupSnapshotDriver))
	Expect(ready).To(BeTrue())
}

func getVolumeGroupSnapshotMembers(groupSnapshot, content *unstructured.Unstructured, expectedCount int) ([]volumeGroupSnapshotMember, error) {
	memberInfo, found, err := unstructured.NestedSlice(content.Object, "status", "volumeSnapshotInfoList")
	if err != nil {
		return nil, err
	}
	if !found {
		memberInfo, found, err = unstructured.NestedSlice(content.Object, "status", "volumeSnapshotHandlePairList")
		if err != nil {
			return nil, err
		}
	}
	if !found {
		return nil, fmt.Errorf("VolumeGroupSnapshotContent %s has no member snapshot list", content.GetName())
	}
	if len(memberInfo) != expectedCount {
		return nil, fmt.Errorf("VolumeGroupSnapshotContent %s has %d member snapshots, expected %d", content.GetName(), len(memberInfo), expectedCount)
	}

	members := make([]volumeGroupSnapshotMember, 0, len(memberInfo))
	for _, item := range memberInfo {
		info, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("VolumeGroupSnapshotContent %s has malformed member snapshot information", content.GetName())
		}
		volumeHandle, _ := info["volumeHandle"].(string)
		snapshotHandle, _ := info["snapshotHandle"].(string)
		if volumeHandle == "" || snapshotHandle == "" {
			return nil, fmt.Errorf("VolumeGroupSnapshotContent %s has a member without a volume or snapshot handle", content.GetName())
		}
		if ready, present := info["readyToUse"].(bool); present && !ready {
			return nil, fmt.Errorf("member snapshot for volume %s is not ready to use", volumeHandle)
		}

		snapshotName := fmt.Sprintf("snapshot-%x", sha256.Sum256([]byte(string(groupSnapshot.GetUID())+volumeHandle)))
		members = append(members, volumeGroupSnapshotMember{volumeHandle: volumeHandle, snapshotName: snapshotName})
	}
	return members, nil
}

func waitForVolumeSnapshotReady(dynamicClient dynamic.Interface, resource schema.GroupVersionResource, namespace, name, groupSnapshotName string) error {
	return waitForResourceCondition(func() (bool, error) {
		snapshot, err := dynamicClient.Resource(resource).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		memberOf, _, err := unstructured.NestedString(snapshot.Object, "spec", "source", "volumeGroupSnapshotName")
		if err != nil {
			return false, err
		}
		if memberOf != groupSnapshotName {
			return false, fmt.Errorf("VolumeSnapshot %s belongs to VolumeGroupSnapshot %q, expected %q", name, memberOf, groupSnapshotName)
		}
		ready, _, err := unstructured.NestedBool(snapshot.Object, "status", "readyToUse")
		return ready, err
	})
}

func waitForVolumeGroupSnapshotDeletion(dynamicClient dynamic.Interface, resources volumeGroupSnapshotResources, namespace, groupSnapshotName, contentName string, members []volumeGroupSnapshotMember) error {
	return waitForResourceCondition(func() (bool, error) {
		_, err := dynamicClient.Resource(resources.groupSnapshot).Namespace(namespace).Get(context.Background(), groupSnapshotName, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}
		if !apierrs.IsNotFound(err) {
			return false, err
		}

		if contentName != "" {
			_, err = dynamicClient.Resource(resources.groupSnapshotContent).Get(context.Background(), contentName, metav1.GetOptions{})
			if err == nil {
				return false, nil
			}
			if !apierrs.IsNotFound(err) {
				return false, err
			}
		}

		for _, member := range members {
			_, err = dynamicClient.Resource(resources.volumeSnapshot).Namespace(namespace).Get(context.Background(), member.snapshotName, metav1.GetOptions{})
			if err == nil {
				return false, nil
			}
			if !apierrs.IsNotFound(err) {
				return false, err
			}
		}
		return true, nil
	})
}

func waitForResourceCondition(condition func() (bool, error)) error {
	return wait.PollUntilContextTimeout(context.Background(), volumeGroupSnapshotPoll, volumeGroupSnapshotTimeout, true, func(context.Context) (bool, error) {
		return condition()
	})
}
