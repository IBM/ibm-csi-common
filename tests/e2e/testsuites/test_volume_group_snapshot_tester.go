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
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	volumeGroupSnapshotAPIGroup = "groupsnapshot.storage.k8s.io"
	volumeGroupSnapshotVersion  = "v1"
	volumeGroupSnapshotDriver   = "vpc.block.csi.ibm.io"
	volumeGroupSnapshotLabelKey = "vgs-e2e.ibm.com/group"
	volumeGroupSnapshotPoll     = 5 * time.Second
	volumeGroupSnapshotTimeout  = 15 * time.Minute
	vgsErrorPollTimeout         = 3 * time.Minute
)

// DynamicallyProvisionedVolumeGroupSnapshotTest runs VGS scenarios using v1 CRDs.
type DynamicallyProvisionedVolumeGroupSnapshotTest struct {
	Volumes []VolumeDetails
}

type volumeGroupSnapshotResources struct {
	apiVersion                                              string
	groupSnapshot, groupSnapshotClass, groupSnapshotContent schema.GroupVersionResource
	volumeSnapshot, volumeSnapshotContent                   schema.GroupVersionResource
}

func vgsResources() volumeGroupSnapshotResources {
	return volumeGroupSnapshotResources{
		apiVersion:            volumeGroupSnapshotAPIGroup + "/" + volumeGroupSnapshotVersion,
		groupSnapshot:         schema.GroupVersionResource{Group: volumeGroupSnapshotAPIGroup, Version: volumeGroupSnapshotVersion, Resource: "volumegroupsnapshots"},
		groupSnapshotClass:    schema.GroupVersionResource{Group: volumeGroupSnapshotAPIGroup, Version: volumeGroupSnapshotVersion, Resource: "volumegroupsnapshotclasses"},
		groupSnapshotContent:  schema.GroupVersionResource{Group: volumeGroupSnapshotAPIGroup, Version: volumeGroupSnapshotVersion, Resource: "volumegroupsnapshotcontents"},
		volumeSnapshot:        schema.GroupVersionResource{Group: SnapshotAPIGroup, Version: APIVersionv1, Resource: "volumesnapshots"},
		volumeSnapshotContent: schema.GroupVersionResource{Group: SnapshotAPIGroup, Version: APIVersionv1, Resource: "volumesnapshotcontents"},
	}
}

type volumeGroupSnapshotMember struct {
	volumeHandle, snapshotHandle, snapshotName string
}

// A record survives readiness failures and remembers generated resources even if
// the controller creates them before the test can read the VGS status.
type vgsRecord struct {
	client                              dynamic.Interface
	resources                           volumeGroupSnapshotResources
	namespace, name                     string
	uid                                 types.UID
	contents, members, snapshotContents map[string]bool
}

func newVGSRecord(client dynamic.Interface, snapshot *unstructured.Unstructured, volumeHandles []string) *vgsRecord {
	r := &vgsRecord{client: client, resources: vgsResources(), namespace: snapshot.GetNamespace(), name: snapshot.GetName(), uid: snapshot.GetUID(),
		contents: map[string]bool{}, members: map[string]bool{}, snapshotContents: map[string]bool{}}
	for _, handle := range volumeHandles {
		r.members[vgsMemberName(r.uid, handle)] = true
	}
	return r
}

func vgsMemberName(uid types.UID, volumeHandle string) string {
	return fmt.Sprintf("snapshot-%x", sha256.Sum256([]byte(string(uid)+volumeHandle)))
}

// observe discovers only resources belonging to this test's VGS. It also returns
// whether any generated resources still exist, including member contents.
func (r *vgsRecord) observe(ctx context.Context) (bool, error) {
	remaining := false
	contents, err := r.client.Resource(r.resources.groupSnapshotContent).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, content := range contents.Items {
		name, _, _ := unstructured.NestedString(content.Object, "spec", "volumeGroupSnapshotRef", "name")
		ns, _, _ := unstructured.NestedString(content.Object, "spec", "volumeGroupSnapshotRef", "namespace")
		uid, _, _ := unstructured.NestedString(content.Object, "spec", "volumeGroupSnapshotRef", "uid")
		if r.contents[content.GetName()] || (name == r.name && ns == r.namespace && uid == string(r.uid)) {
			r.contents[content.GetName()] = true
			remaining = true
		}
	}
	snapshots, err := r.client.Resource(r.resources.volumeSnapshot).Namespace(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, snapshot := range snapshots.Items {
		owned := r.members[snapshot.GetName()]
		for _, owner := range snapshot.GetOwnerReferences() {
			if owner.Kind == "VolumeGroupSnapshot" && owner.UID == r.uid {
				owned = true
			}
		}
		if owned {
			r.members[snapshot.GetName()] = true
			remaining = true
			name, _, _ := unstructured.NestedString(snapshot.Object, "status", "boundVolumeSnapshotContentName")
			if name != "" {
				r.snapshotContents[name] = true
			}
		}
	}
	snapshotContents, err := r.client.Resource(r.resources.volumeSnapshotContent).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, content := range snapshotContents.Items {
		name, _, _ := unstructured.NestedString(content.Object, "spec", "volumeSnapshotRef", "name")
		ns, _, _ := unstructured.NestedString(content.Object, "spec", "volumeSnapshotRef", "namespace")
		if r.snapshotContents[content.GetName()] || (ns == r.namespace && r.members[name]) {
			r.snapshotContents[content.GetName()] = true
			remaining = true
		}
	}
	return remaining, nil
}

func (r *vgsRecord) deleteAndWait(ctx context.Context, poll, timeout time.Duration) error {
	// Capture content/member names before requesting deletion, including partial setup.
	if _, err := r.observe(ctx); err != nil {
		return err
	}
	// Members cannot drop their in-group finalizer until the VGS is gone.
	// Foreground propagation would keep that VGS around and deadlock deletion.
	options := metav1.DeleteOptions{PropagationPolicy: ptrTo(metav1.DeletePropagationBackground)}
	if r.uid != "" {
		options.Preconditions = &metav1.Preconditions{UID: &r.uid}
	}
	if err := r.client.Resource(r.resources.groupSnapshot).Namespace(r.namespace).Delete(ctx, r.name, options); err != nil && !apierrs.IsNotFound(err) {
		return err
	}
	err := wait.PollUntilContextTimeout(ctx, poll, timeout, true, func(ctx context.Context) (bool, error) {
		_, err := r.client.Resource(r.resources.groupSnapshot).Namespace(r.namespace).Get(ctx, r.name, metav1.GetOptions{})
		if err != nil && !apierrs.IsNotFound(err) {
			return false, err
		}
		remaining, observeErr := r.observe(ctx)
		return apierrs.IsNotFound(err) && !remaining, observeErr
	})
	if err != nil {
		return fmt.Errorf("cleanup of VGS %s/%s (contents=%v, members=%v, member contents=%v): %w", r.namespace, r.name, r.contents, r.members, r.snapshotContents, err)
	}
	return nil
}

type vgsClaim struct {
	pvc      *TestPersistentVolumeClaim
	created  *v1.PersistentVolumeClaim
	restored bool
}

func (claim vgsClaim) cleanup(ctx context.Context, client clientset.Interface) error {
	// WaitForBound may fail during its final Get. Preserve the originally created
	// claim identity even if that helper replaces its live object with nil.
	pvc := *claim.pvc
	if pvc.persistentVolumeClaim == nil {
		pvc.persistentVolumeClaim = claim.created
	}
	return deleteVGSClaim(ctx, client, &pvc)
}

type vgsFixture struct {
	client    clientset.Interface
	dynamic   dynamic.Interface
	namespace *v1.Namespace
	resources volumeGroupSnapshotResources
	claims    []vgsClaim
	pods      []*TestPod
	groups    []*vgsRecord
	classes   []string
}

func newVGSFixture(client clientset.Interface, dc dynamic.Interface, ns *v1.Namespace) *vgsFixture {
	return &vgsFixture{client: client, dynamic: dc, namespace: ns, resources: vgsResources()}
}

func (f *vgsFixture) prerequisites() {
	By("checking access to the required v1 snapshot APIs")
	for _, gvr := range []schema.GroupVersionResource{f.resources.groupSnapshot, f.resources.groupSnapshotClass, f.resources.groupSnapshotContent, f.resources.volumeSnapshot, f.resources.volumeSnapshotContent} {
		var err error
		if gvr == f.resources.groupSnapshot || gvr == f.resources.volumeSnapshot {
			_, err = f.dynamic.Resource(gvr).Namespace(f.namespace.Name).List(context.Background(), metav1.ListOptions{Limit: 1})
		} else {
			_, err = f.dynamic.Resource(gvr).List(context.Background(), metav1.ListOptions{Limit: 1})
		}
		framework.ExpectNoError(err, "the supplied cluster must serve %s and allow the E2E account to access it", gvr.String())
	}
}

func (f *vgsFixture) provision(volume VolumeDetails, selected, restored bool) *TestPersistentVolumeClaim {
	sc, err := f.client.StorageV1().StorageClasses().Get(context.Background(), volume.VolumeType, metav1.GetOptions{})
	framework.ExpectNoError(err)
	if sc.VolumeBindingMode != nil && string(*sc.VolumeBindingMode) != "Immediate" {
		Fail("VGS E2E requires an Immediate-binding storage class so unattached PVCs can be tested")
	}
	Expect(sc.ReclaimPolicy).NotTo(BeNil())
	Expect(*sc.ReclaimPolicy).To(Equal(v1.PersistentVolumeReclaimDelete), "VGS E2E requires Delete-reclaim source and restore volumes")
	var pvc *TestPersistentVolumeClaim
	if volume.DataSource == nil {
		pvc = NewTestPersistentVolumeClaim(f.client, volume.PVCName, f.namespace, volume.ClaimSize, volume.AccessMode, volume.VolumeMode, sc)
	} else {
		pvc = NewTestPersistentVolumeClaimWithDataSource(f.client, volume.PVCName, f.namespace, volume.ClaimSize, volume.VolumeMode, sc,
			&v1.TypedLocalObjectReference{APIGroup: &SnapshotAPIGroup, Kind: VolumeSnapshotKind, Name: volume.DataSource.Name})
	}
	pvc.Create()
	// Register before binding/validation can fail.
	f.claims = append(f.claims, vgsClaim{pvc: pvc, created: pvc.persistentVolumeClaim, restored: restored})
	pvc.WaitForBound()
	pvc.ValidateProvisionedPersistentVolume()
	Expect(pvc.persistentVolume.Spec.CSI).NotTo(BeNil())
	Expect(pvc.persistentVolume.Spec.CSI.Driver).To(Equal(volumeGroupSnapshotDriver))
	Expect(pvc.persistentVolume.Spec.CSI.VolumeHandle).NotTo(BeEmpty())
	if selected {
		claim := pvc.persistentVolumeClaim.DeepCopy()
		if claim.Labels == nil {
			claim.Labels = map[string]string{}
		}
		claim.Labels[volumeGroupSnapshotLabelKey] = f.namespace.Name
		updated, err := f.client.CoreV1().PersistentVolumeClaims(f.namespace.Name).Update(context.Background(), claim, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
		pvc.persistentVolumeClaim = updated
	}
	return pvc
}

func (f *vgsFixture) mount(pvcs []*TestPersistentVolumeClaim) *TestPod {
	pod := NewTestPod(f.client, f.namespace, "while true; do sleep 2; done")
	for i, pvc := range pvcs {
		pod.SetupVolume(pvc.persistentVolumeClaim, fmt.Sprintf("vgs-volume-%d", i+1), fmt.Sprintf("/mnt/vgs-%d", i+1), false)
	}
	pod.Create()
	f.pods = append(f.pods, pod)
	pod.WaitForRunningSlow()
	return pod
}

func vgsCheckData(pod *TestPod, data []string, write bool) {
	for i, value := range data {
		path := fmt.Sprintf("/mnt/vgs-%d/data", i+1)
		pod.Exec([]string{"/bin/sh", "-c", vgsDataCommand(path, value, write)}, "VGS_DATA_OK")
	}
}

func vgsDataCommand(path, value string, write bool) string {
	command := fmt.Sprintf("test \"$(cat %s)\" = %q && echo VGS_DATA_OK", path, value)
	if write {
		command = fmt.Sprintf("printf '%%s\n' %q > %s && sync && ", value, path) + command
	}
	return command
}

func (f *vgsFixture) createGroup(handles []string, selector, className string) *vgsRecord {
	if className == "" {
		class, err := f.dynamic.Resource(f.resources.groupSnapshotClass).Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": f.resources.apiVersion, "kind": "VolumeGroupSnapshotClass", "metadata": map[string]interface{}{"generateName": f.namespace.Name + "-vgs-class-"},
			"driver": volumeGroupSnapshotDriver, "deletionPolicy": "Delete",
		}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		className = class.GetName()
		f.classes = append(f.classes, className)
	}
	snapshot, err := f.dynamic.Resource(f.resources.groupSnapshot).Namespace(f.namespace.Name).Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": f.resources.apiVersion, "kind": "VolumeGroupSnapshot", "metadata": map[string]interface{}{"generateName": "vgs-e2e-", "namespace": f.namespace.Name},
		"spec": map[string]interface{}{"volumeGroupSnapshotClassName": className, "source": map[string]interface{}{"selector": map[string]interface{}{"matchLabels": map[string]interface{}{volumeGroupSnapshotLabelKey: selector}}}},
	}}, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	r := newVGSRecord(f.dynamic, snapshot, handles)
	f.groups = append(f.groups, r)
	return r
}

func (f *vgsFixture) ready(r *vgsRecord, handles []string) []volumeGroupSnapshotMember {
	snapshot, err := waitForVGSReady(context.Background(), f.dynamic, r.namespace, r.name, volumeGroupSnapshotPoll, volumeGroupSnapshotTimeout)
	framework.ExpectNoError(err)
	contentName, _, err := unstructured.NestedString(snapshot.Object, "status", "boundVolumeGroupSnapshotContentName")
	framework.ExpectNoError(err)
	Expect(contentName).NotTo(BeEmpty())
	r.contents[contentName] = true
	content, err := f.dynamic.Resource(f.resources.groupSnapshotContent).Get(context.Background(), contentName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	framework.ExpectNoError(validateVGSBinding(snapshot, content))
	members, err := getVolumeGroupSnapshotMembers(snapshot, content, handles)
	framework.ExpectNoError(err)
	return members
}

func (f *vgsFixture) restore(volumes []VolumeDetails, members []volumeGroupSnapshotMember, data []string) {
	By("restoring both member snapshots and verifying the saved data")
	byHandle := map[string]string{}
	for _, member := range members {
		byHandle[member.volumeHandle] = member.snapshotName
	}
	var restored []*TestPersistentVolumeClaim
	for i, claim := range f.claims {
		if claim.restored || len(restored) == len(volumes) {
			break
		}
		handle := claim.pvc.persistentVolume.Spec.CSI.VolumeHandle
		volume := volumes[i]
		volume.PVCName = fmt.Sprintf("ics-vgs-restored-%d-", i+1)
		Expect(byHandle[handle]).NotTo(BeEmpty())
		volume.DataSource = &DataSource{Name: byHandle[handle]}
		restored = append(restored, f.provision(volume, false, true))
	}
	vgsCheckData(f.mount(restored), data, false)
}

func deleteVGSPod(ctx context.Context, client clientset.Interface, pod *TestPod) error {
	if pod.pod == nil || pod.pod.Name == "" {
		return nil
	}
	ns, name := pod.namespace.Name, pod.pod.Name
	err := client.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		return err
	}
	return wait.PollUntilContextTimeout(ctx, volumeGroupSnapshotPoll, volumeGroupSnapshotTimeout, true, func(ctx context.Context) (bool, error) {
		_, err := client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func deleteVGSClaim(ctx context.Context, client clientset.Interface, pvc *TestPersistentVolumeClaim) error {
	if pvc.persistentVolumeClaim == nil {
		return nil
	}
	ns, name := pvc.namespace.Name, pvc.persistentVolumeClaim.Name
	pvName := pvc.persistentVolumeClaim.Spec.VolumeName
	claim, err := client.CoreV1().PersistentVolumeClaims(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		return err
	}
	if err == nil {
		pvName = claim.Spec.VolumeName
	}
	if err := client.CoreV1().PersistentVolumeClaims(ns).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrs.IsNotFound(err) {
		return err
	}
	return wait.PollUntilContextTimeout(ctx, volumeGroupSnapshotPoll, volumeGroupSnapshotTimeout, true, func(ctx context.Context) (bool, error) {
		_, err := client.CoreV1().PersistentVolumeClaims(ns).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}
		if !apierrs.IsNotFound(err) {
			return false, err
		}
		if pvName != "" {
			_, err = client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
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

func (f *vgsFixture) cleanup() {
	if CurrentSpecReport().Failed() {
		f.diagnostics()
	}
	framework.ExpectNoError(f.cleanupResources(context.Background(), volumeGroupSnapshotPoll, volumeGroupSnapshotTimeout), "VGS test resource cleanup failed")
}

func (f *vgsFixture) cleanupResources(ctx context.Context, poll, timeout time.Duration) error {
	var errs []error
	for i := len(f.pods) - 1; i >= 0; i-- {
		errs = append(errs, deleteVGSPod(ctx, f.client, f.pods[i]))
	}
	// Pending restore PVCs protect their source snapshots, so remove them first.
	for _, claim := range f.claims {
		if claim.restored {
			errs = append(errs, claim.cleanup(ctx, f.client))
		}
	}
	for _, group := range f.groups {
		errs = append(errs, group.deleteAndWait(ctx, poll, timeout))
	}
	for _, claim := range f.claims {
		if !claim.restored {
			errs = append(errs, claim.cleanup(ctx, f.client))
		}
	}
	for _, name := range f.classes {
		err := f.dynamic.Resource(f.resources.groupSnapshotClass).Delete(ctx, name, metav1.DeleteOptions{})
		if !apierrs.IsNotFound(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (f *vgsFixture) diagnostics() {
	for _, group := range f.groups {
		obj, err := f.dynamic.Resource(f.resources.groupSnapshot).Namespace(group.namespace).Get(context.Background(), group.name, metav1.GetOptions{})
		if err == nil {
			framework.Logf("VGS %s status: %v", group.name, obj.Object["status"])
		} else {
			framework.Logf("VGS %s: %v", group.name, err)
		}
	}
	events, err := f.client.CoreV1().Events(f.namespace.Name).List(context.Background(), metav1.ListOptions{})
	if err == nil {
		for _, event := range events.Items {
			framework.Logf("VGS E2E event %s/%s %s: %s", event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Reason, event.Message)
		}
	}
}

func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) runPositive(client clientset.Interface, dc dynamic.Interface, ns *v1.Namespace, scenario string) {
	f := newVGSFixture(client, dc, ns)
	defer f.cleanup()
	f.prerequisites()
	count := 2
	if scenario == "selector" {
		count = 3
	}
	Expect(t.Volumes).To(HaveLen(count))
	var pvcs []*TestPersistentVolumeClaim
	var handles, data []string
	for i, volume := range t.Volumes {
		pvc := f.provision(volume, i < 2, false)
		if i < 2 {
			pvcs = append(pvcs, pvc)
			handles = append(handles, pvc.persistentVolume.Spec.CSI.VolumeHandle)
			data = append(data, fmt.Sprintf("vgs-before-%d", i+1))
		}
	}
	sourcePod := f.mount(pvcs)
	vgsCheckData(sourcePod, data, true)
	r := f.createGroup(handles, ns.Name, "")
	if scenario == "creating" {
		By("observing bound snapshot content before deleting the in-progress request")
		err := wait.PollUntilContextTimeout(context.Background(), 250*time.Millisecond, volumeGroupSnapshotTimeout, true, func(ctx context.Context) (bool, error) {
			obj, err := dc.Resource(f.resources.groupSnapshot).Namespace(ns.Name).Get(ctx, r.name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			ready, _, _ := unstructured.NestedBool(obj.Object, "status", "readyToUse")
			if ready {
				return false, errVGSCreationAlreadyReady
			}
			name, _, err := unstructured.NestedString(obj.Object, "status", "boundVolumeGroupSnapshotContentName")
			if name == "" {
				return false, err
			}
			content, err := dc.Resource(f.resources.groupSnapshotContent).Get(ctx, name, metav1.GetOptions{})
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			contentReady, _, _ := unstructured.NestedBool(content.Object, "status", "readyToUse")
			if contentReady {
				return false, errVGSCreationAlreadyReady
			}
			r.contents[name] = true
			return true, nil
		})
		if errors.Is(err, errVGSCreationAlreadyReady) {
			Skip("VGS became ready before an in-progress creation could be observed; rerun this scenario")
		}
		framework.ExpectNoError(err)
		framework.ExpectNoError(r.deleteAndWait(context.Background(), volumeGroupSnapshotPoll, volumeGroupSnapshotTimeout))
		return
	}
	members := f.ready(r, handles)
	switch scenario {
	case "selector":
		// Exact handle matching in ready() also proves the unlabeled third PVC was excluded.
		return
	case "point-in-time":
		By("executing and verifying new writes after snapshot creation")
		vgsCheckData(sourcePod, []string{"vgs-after-1", "vgs-after-2"}, true)
	case "source-delete":
		By("deleting the source pod, PVCs, and PVs before restoring")
		framework.ExpectNoError(deleteVGSPod(context.Background(), client, sourcePod))
		for _, pvc := range pvcs {
			framework.ExpectNoError(deleteVGSClaim(context.Background(), client, pvc))
		}
	case "repeat-delete":
		framework.ExpectNoError(r.deleteAndWait(context.Background(), volumeGroupSnapshotPoll, volumeGroupSnapshotTimeout))
		err := dc.Resource(f.resources.groupSnapshot).Namespace(ns.Name).Delete(context.Background(), r.name, metav1.DeleteOptions{})
		Expect(apierrs.IsNotFound(err)).To(BeTrue(), "a repeated Kubernetes delete must report NotFound, got %v", err)
		return
	}
	f.restore(t.Volumes, members, data)
}

var errVGSCreationAlreadyReady = errors.New("VGS creation already completed")

func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) Run(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runPositive(c, d, n, "basic")
}
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunSelectorIsolation(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runPositive(c, d, n, "selector")
}
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunPointInTimeRestore(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runPositive(c, d, n, "point-in-time")
}
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunRestoreAfterSourceDelete(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runPositive(c, d, n, "source-delete")
}
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunDeleteWhileCreating(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runPositive(c, d, n, "creating")
}

// RunDeleteIdempotency checks repeated Kubernetes deletion, not CSI RPC idempotency.
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunDeleteIdempotency(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runPositive(c, d, n, "repeat-delete")
}

func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) runNegative(client clientset.Interface, dc dynamic.Interface, ns *v1.Namespace, kind string) {
	f := newVGSFixture(client, dc, ns)
	defer f.cleanup()
	f.prerequisites()
	var handles []string
	if kind != "no-pvcs" {
		Expect(t.Volumes).To(HaveLen(2))
		for _, volume := range t.Volumes {
			pvc := f.provision(volume, true, false)
			handles = append(handles, pvc.persistentVolume.Spec.CSI.VolumeHandle)
		}
	}
	selector, class := ns.Name, ""
	if kind == "no-pvcs" {
		selector += "-unused"
	}
	if kind == "invalid-class" {
		class = ns.Name + "-missing"
		_, err := dc.Resource(f.resources.groupSnapshotClass).Get(context.Background(), class, metav1.GetOptions{})
		Expect(apierrs.IsNotFound(err)).To(BeTrue(), "the negative test's class name must not exist")
	}
	if kind == "unattached" {
		attachments, err := client.StorageV1().VolumeAttachments().List(context.Background(), metav1.ListOptions{})
		framework.ExpectNoError(err)
		for _, attachment := range attachments.Items {
			for _, claim := range f.claims {
				if attachment.Spec.Source.PersistentVolumeName != nil {
					Expect(*attachment.Spec.Source.PersistentVolumeName).NotTo(Equal(claim.pvc.persistentVolume.Name), "negative test source volumes must be unattached")
				}
			}
		}
	}
	r := f.createGroup(handles, selector, class)
	framework.ExpectNoError(waitForVGSError(context.Background(), dc, ns.Name, r.name, kind, class, volumeGroupSnapshotPoll, vgsErrorPollTimeout))
}

func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunNoPVCs(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runNegative(c, d, n, "no-pvcs")
}
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunUnattachedVolumes(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runNegative(c, d, n, "unattached")
}
func (t *DynamicallyProvisionedVolumeGroupSnapshotTest) RunInvalidClass(c clientset.Interface, d dynamic.Interface, n *v1.Namespace) {
	t.runNegative(c, d, n, "invalid-class")
}

func waitForVGSReady(ctx context.Context, dc dynamic.Interface, ns, name string, poll, timeout time.Duration) (*unstructured.Unstructured, error) {
	var result *unstructured.Unstructured
	lastStatus := interface{}(nil)
	err := wait.PollUntilContextTimeout(ctx, poll, timeout, true, func(ctx context.Context) (bool, error) {
		obj, err := dc.Resource(vgsResources().groupSnapshot).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		lastStatus = obj.Object["status"]
		ready, _, err := unstructured.NestedBool(obj.Object, "status", "readyToUse")
		if ready && err == nil {
			result = obj
		}
		return ready, err
	})
	if err != nil {
		return nil, fmt.Errorf("VGS %s/%s did not become ready; last status=%v: %w", ns, name, lastStatus, err)
	}
	return result, nil
}

var (
	vgsNoPVCError          = regexp.MustCompile(`(?i)\b(no|zero|0)\s+(matching\s+)?(pvcs?|persistent[ -]?volume[ -]?claims?|volumes?)\b|\b(pvcs?|persistent[ -]?volume[ -]?claims?|volumes?)\s+(list\s+)?(is\s+)?empty\b|\bfind any\s+(pvcs?|volumes?)\b`)
	vgsUnattachedError     = regexp.MustCompile(`(?i)\bunattached\b|not attached|must be attached|\bdetached\b|attachment.*(missing|not found)|volume.*(\bavailable\b|\bin[_ -]use\b)`)
	vgsMissingClassError   = regexp.MustCompile(`(?i)not found|does not exist|missing`)
	vgsInfrastructureError = regexp.MustCompile(`(?i)unauthorized|unauthenticated|forbidden|permission denied|credentials|connection refused|deadline exceeded|unimplemented|unsupported|not supported|service unavailable|timed out`)
)

func expectedVGSError(kind, className, message string) bool {
	if vgsInfrastructureError.MatchString(message) {
		return false
	}
	switch kind {
	case "no-pvcs":
		return vgsNoPVCError.MatchString(message)
	case "unattached":
		return vgsUnattachedError.MatchString(message)
	case "invalid-class":
		return strings.Contains(message, className) && strings.Contains(strings.ToLower(message), "class") && vgsMissingClassError.MatchString(message)
	}
	return false
}

func waitForVGSError(ctx context.Context, dc dynamic.Interface, ns, name, kind, className string, poll, timeout time.Duration) error {
	lastMessage := ""
	err := wait.PollUntilContextTimeout(ctx, poll, timeout, true, func(ctx context.Context) (bool, error) {
		obj, err := dc.Resource(vgsResources().groupSnapshot).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		ready, _, err := unstructured.NestedBool(obj.Object, "status", "readyToUse")
		if err != nil {
			return false, err
		}
		if ready {
			return false, fmt.Errorf("negative VGS scenario %s unexpectedly became ready", kind)
		}
		lastMessage, _, err = unstructured.NestedString(obj.Object, "status", "error", "message")
		if err != nil {
			return false, err
		}
		return expectedVGSError(kind, className, lastMessage), nil
	})
	if err != nil {
		return fmt.Errorf("VGS %s/%s did not report the expected %s error (last message=%q): %w", ns, name, kind, lastMessage, err)
	}
	return nil
}

func validateVGSBinding(snapshot, content *unstructured.Unstructured) error {
	checks := []struct {
		object *unstructured.Unstructured
		path   []string
		want   string
	}{
		{snapshot, []string{"status", "boundVolumeGroupSnapshotContentName"}, content.GetName()},
		{content, []string{"spec", "volumeGroupSnapshotRef", "name"}, snapshot.GetName()},
		{content, []string{"spec", "volumeGroupSnapshotRef", "namespace"}, snapshot.GetNamespace()},
		{content, []string{"spec", "volumeGroupSnapshotRef", "uid"}, string(snapshot.GetUID())},
		{content, []string{"spec", "driver"}, volumeGroupSnapshotDriver},
		{content, []string{"spec", "deletionPolicy"}, "Delete"},
	}
	if snapshot.GetUID() == "" {
		return errors.New("VGS has no UID")
	}
	for _, check := range checks {
		value, found, err := unstructured.NestedString(check.object.Object, check.path...)
		if err != nil {
			return err
		}
		if !found || value != check.want {
			return fmt.Errorf("%s: %s = %q, expected %q", check.object.GetName(), strings.Join(check.path, "."), value, check.want)
		}
	}
	for _, object := range []*unstructured.Unstructured{snapshot, content} {
		ready, _, err := unstructured.NestedBool(object.Object, "status", "readyToUse")
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("%s is not ready", object.GetName())
		}
	}
	handle, _, err := unstructured.NestedString(content.Object, "status", "volumeGroupSnapshotHandle")
	if err != nil {
		return err
	}
	if handle == "" {
		return errors.New("VGS content has no backend group snapshot handle")
	}
	return nil
}

func getVolumeGroupSnapshotMembers(snapshot, content *unstructured.Unstructured, expectedHandles []string) ([]volumeGroupSnapshotMember, error) {
	if snapshot.GetUID() == "" {
		return nil, errors.New("VGS has no UID for identifying its member snapshots")
	}
	info, found, err := unstructured.NestedSlice(content.Object, "status", "volumeSnapshotInfoList")
	if err != nil {
		return nil, err
	}
	if !found || len(info) != len(expectedHandles) {
		return nil, fmt.Errorf("content %s has %d member snapshots, expected %d", content.GetName(), len(info), len(expectedHandles))
	}
	expected, seenSnapshots := map[string]bool{}, map[string]bool{}
	for _, handle := range expectedHandles {
		if handle == "" || expected[handle] {
			return nil, errors.New("expected source volume handles must be nonempty and unique")
		}
		expected[handle] = true
	}
	var members []volumeGroupSnapshotMember
	for _, item := range info {
		entry, ok := item.(map[string]interface{})
		if !ok {
			return nil, errors.New("malformed VGS member information")
		}
		volume, _ := entry["volumeHandle"].(string)
		handle, _ := entry["snapshotHandle"].(string)
		if !expected[volume] || handle == "" || seenSnapshots[handle] {
			return nil, fmt.Errorf("unexpected/duplicate volume or snapshot handle in member %v", entry)
		}
		delete(expected, volume)
		seenSnapshots[handle] = true
		members = append(members, volumeGroupSnapshotMember{volumeHandle: volume, snapshotHandle: handle, snapshotName: vgsMemberName(snapshot.GetUID(), volume)})
	}
	return members, nil
}

func ptrTo[T any](value T) *T { return &value }
