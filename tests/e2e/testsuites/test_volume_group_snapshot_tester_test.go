/**
 * Copyright 2026 IBM Corp.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package testsuites

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

const vgsUnitNamespace = "vgs-unit"

func TestVGSDataCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data")
	for _, value := range []string{"vgs-before-1", "vgs-after-1"} {
		output, err := exec.Command("/bin/sh", "-c", vgsDataCommand(path, value, true)).CombinedOutput()
		if err != nil || !strings.Contains(string(output), "VGS_DATA_OK") {
			t.Fatalf("write/verify failed: %s, %v", output, err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != value+"\n" {
			t.Fatalf("on-disk data=%q, err=%v", data, err)
		}
	}
	if err := exec.Command("/bin/sh", "-c", vgsDataCommand(path, "vgs-before-1", false)).Run(); err == nil {
		t.Fatal("reading overwritten data must not pass an old-data check")
	}
}

func unitVGS() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": vgsResources().apiVersion, "kind": "VolumeGroupSnapshot",
		"metadata": map[string]interface{}{"name": "group-1", "namespace": vgsUnitNamespace, "uid": "group-uid"},
		"status":   map[string]interface{}{"readyToUse": true, "boundVolumeGroupSnapshotContentName": "group-content"},
	}}
}

func unitVGSContent() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": vgsResources().apiVersion, "kind": "VolumeGroupSnapshotContent",
		"metadata": map[string]interface{}{"name": "group-content"},
		"spec": map[string]interface{}{"driver": volumeGroupSnapshotDriver, "deletionPolicy": "Delete",
			"volumeGroupSnapshotRef": map[string]interface{}{"name": "group-1", "namespace": vgsUnitNamespace, "uid": "group-uid"}},
		"status": map[string]interface{}{"readyToUse": true, "volumeGroupSnapshotHandle": "backend-group",
			"volumeSnapshotInfoList": []interface{}{
				map[string]interface{}{"volumeHandle": "vol-1", "snapshotHandle": "snap-1"},
				map[string]interface{}{"volumeHandle": "vol-2", "snapshotHandle": "snap-2"},
			}},
	}}
}

func unitDynamic(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	r := vgsResources()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		r.groupSnapshot: "VolumeGroupSnapshotList", r.groupSnapshotContent: "VolumeGroupSnapshotContentList",
		r.groupSnapshotClass: "VolumeGroupSnapshotClassList", r.volumeSnapshot: "VolumeSnapshotList", r.volumeSnapshotContent: "VolumeSnapshotContentList",
	}, objects...)
}

func TestVGSUsesV1(t *testing.T) {
	r := vgsResources()
	for _, gvr := range []schema.GroupVersionResource{r.groupSnapshot, r.groupSnapshotClass, r.groupSnapshotContent, r.volumeSnapshot, r.volumeSnapshotContent} {
		if gvr.Version != "v1" {
			t.Fatalf("expected v1, got %s", gvr)
		}
	}
}

func TestValidateVGSBinding(t *testing.T) {
	if err := validateVGSBinding(unitVGS(), unitVGSContent()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		snapshot bool
		path     []string
		value    interface{}
	}{
		{"wrong bound content", true, []string{"status", "boundVolumeGroupSnapshotContentName"}, "other"},
		{"wrong reference name", false, []string{"spec", "volumeGroupSnapshotRef", "name"}, "other"},
		{"wrong namespace", false, []string{"spec", "volumeGroupSnapshotRef", "namespace"}, "other"},
		{"wrong UID", false, []string{"spec", "volumeGroupSnapshotRef", "uid"}, "old-uid"},
		{"wrong driver", false, []string{"spec", "driver"}, "file.csi.example"},
		{"wrong deletion policy", false, []string{"spec", "deletionPolicy"}, "Retain"},
		{"unready group", true, []string{"status", "readyToUse"}, false},
		{"unready content", false, []string{"status", "readyToUse"}, false},
		{"missing backend handle", false, []string{"status", "volumeGroupSnapshotHandle"}, ""},
		{"missing UID", true, []string{"metadata", "uid"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			group, content := unitVGS(), unitVGSContent()
			obj := content
			if tc.snapshot {
				obj = group
			}
			if err := unstructured.SetNestedField(obj.Object, tc.value, tc.path...); err != nil {
				t.Fatal(err)
			}
			if err := validateVGSBinding(group, content); err == nil {
				t.Fatal("expected binding validation failure")
			}
		})
	}
}

func TestGetVGSMembers(t *testing.T) {
	group, content := unitVGS(), unitVGSContent()
	// Return members in the reverse order: matching must use handles, not position.
	members, err := getVolumeGroupSnapshotMembers(group, content, []string{"vol-2", "vol-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[0].snapshotName != vgsMemberName(group.GetUID(), "vol-1") || members[0].snapshotHandle != "snap-1" {
		t.Fatalf("unexpected members: %+v", members)
	}
	cases := []struct {
		name string
		list []interface{}
	}{
		{"missing member", []interface{}{map[string]interface{}{"volumeHandle": "vol-1", "snapshotHandle": "snap-1"}}},
		{"unselected volume", []interface{}{map[string]interface{}{"volumeHandle": "vol-3", "snapshotHandle": "snap-1"}, map[string]interface{}{"volumeHandle": "vol-2", "snapshotHandle": "snap-2"}}},
		{"duplicate volume", []interface{}{map[string]interface{}{"volumeHandle": "vol-1", "snapshotHandle": "snap-1"}, map[string]interface{}{"volumeHandle": "vol-1", "snapshotHandle": "snap-2"}}},
		{"duplicate snapshot", []interface{}{map[string]interface{}{"volumeHandle": "vol-1", "snapshotHandle": "snap-1"}, map[string]interface{}{"volumeHandle": "vol-2", "snapshotHandle": "snap-1"}}},
		{"empty snapshot", []interface{}{map[string]interface{}{"volumeHandle": "vol-1", "snapshotHandle": ""}, map[string]interface{}{"volumeHandle": "vol-2", "snapshotHandle": "snap-2"}}},
		{"malformed member", []interface{}{"invalid", map[string]interface{}{"volumeHandle": "vol-2", "snapshotHandle": "snap-2"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := content.DeepCopy()
			if err := unstructured.SetNestedSlice(c.Object, tc.list, "status", "volumeSnapshotInfoList"); err != nil {
				t.Fatal(err)
			}
			if _, err := getVolumeGroupSnapshotMembers(group, c, []string{"vol-1", "vol-2"}); err == nil {
				t.Fatal("expected member validation failure")
			}
		})
	}
	t.Run("rejects beta field", func(t *testing.T) {
		c := content.DeepCopy()
		unstructured.RemoveNestedField(c.Object, "status", "volumeSnapshotInfoList")
		if _, err := getVolumeGroupSnapshotMembers(group, c, []string{"vol-1", "vol-2"}); err == nil {
			t.Fatal("expected v1 member list")
		}
	})
	t.Run("rejects duplicate expected source handles", func(t *testing.T) {
		if _, err := getVolumeGroupSnapshotMembers(group, content, []string{"vol-1", "vol-1"}); err == nil {
			t.Fatal("expected invalid source handles to fail")
		}
	})
}

func TestExpectedVGSError(t *testing.T) {
	cases := []struct {
		kind, class, message string
		want                 bool
	}{
		{"no-pvcs", "", "No PVCs found for the selector", true},
		{"no-pvcs", "", "No volumes were found", true},
		{"no-pvcs", "", "failed to update status", false},
		{"no-pvcs", "", "forbidden: no PVCs could be listed", false},
		{"no-pvcs", "", "No group snapshot support for this volume driver", false},
		{"no-pvcs", "", "No default snapshot class found for volumes", false},
		{"unattached", "", "source volume is not attached to an instance", true},
		{"unattached", "", "volume must be in_use", true},
		{"unattached", "", "volume service unavailable", false},
		{"unattached", "", "credentials unavailable for unattached volume", false},
		{"invalid-class", "missing-class", `volumegroupsnapshotclasses "missing-class" not found`, true},
		{"invalid-class", "missing-class", `volumegroupsnapshotclasses "other-class" not found`, false},
		{"invalid-class", "missing-class", `forbidden reading class missing-class`, false},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.message, func(t *testing.T) {
			if got := expectedVGSError(tc.kind, tc.class, tc.message); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestVGSWaits(t *testing.T) {
	t.Run("ready after polling", func(t *testing.T) {
		group := unitVGS()
		client := unitDynamic(group)
		calls := 0
		client.PrependReactor("get", "volumegroupsnapshots", func(clienttesting.Action) (bool, runtime.Object, error) {
			calls++
			obj := group.DeepCopy()
			_ = unstructured.SetNestedField(obj.Object, calls > 1, "status", "readyToUse")
			return true, obj, nil
		})
		got, err := waitForVGSReady(context.Background(), client, vgsUnitNamespace, "group-1", time.Millisecond, time.Second)
		if err != nil || got == nil || calls < 2 {
			t.Fatalf("got %v, error %v, calls %d", got, err, calls)
		}
	})
	t.Run("timeout preserves record for cleanup", func(t *testing.T) {
		group := unitVGS()
		_ = unstructured.SetNestedField(group.Object, false, "status", "readyToUse")
		client := unitDynamic(group)
		record := newVGSRecord(client, group, []string{"vol-1"})
		got, err := waitForVGSReady(context.Background(), client, vgsUnitNamespace, record.name, time.Millisecond, 5*time.Millisecond)
		if err == nil || got != nil || record.name != "group-1" {
			t.Fatalf("got %v, error %v, record %v", got, err, record)
		}
		if err := record.deleteAndWait(context.Background(), time.Millisecond, time.Second); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("negative test rejects unrelated error", func(t *testing.T) {
		group := unitVGS()
		_ = unstructured.SetNestedField(group.Object, false, "status", "readyToUse")
		_ = unstructured.SetNestedField(group.Object, "forbidden listing PVCs", "status", "error", "message")
		err := waitForVGSError(context.Background(), unitDynamic(group), vgsUnitNamespace, "group-1", "no-pvcs", "", time.Millisecond, 5*time.Millisecond)
		if err == nil || !strings.Contains(err.Error(), "forbidden") {
			t.Fatalf("expected diagnostic failure, got %v", err)
		}
	})
	t.Run("negative test requires the expected cause", func(t *testing.T) {
		group := unitVGS()
		_ = unstructured.SetNestedField(group.Object, false, "status", "readyToUse")
		_ = unstructured.SetNestedField(group.Object, "No PVCs matched the selector", "status", "error", "message")
		if err := waitForVGSError(context.Background(), unitDynamic(group), vgsUnitNamespace, "group-1", "no-pvcs", "", time.Millisecond, time.Second); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("negative test rejects success", func(t *testing.T) {
		if err := waitForVGSError(context.Background(), unitDynamic(unitVGS()), vgsUnitNamespace, "group-1", "no-pvcs", "", time.Millisecond, time.Second); err == nil {
			t.Fatal("expected unexpected readiness to fail")
		}
	})
	t.Run("API error is not an expected negative result", func(t *testing.T) {
		client := unitDynamic(unitVGS())
		client.PrependReactor("get", "volumegroupsnapshots", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("permission denied")
		})
		if err := waitForVGSError(context.Background(), client, vgsUnitNamespace, "group-1", "no-pvcs", "", time.Millisecond, time.Second); err == nil {
			t.Fatal("expected API failure")
		}
	})
}

func unitMember() *unstructured.Unstructured {
	group := unitVGS()
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": SnapshotAPIVersion, "kind": "VolumeSnapshot",
		"metadata": map[string]interface{}{"name": vgsMemberName(group.GetUID(), "vol-1"), "namespace": vgsUnitNamespace},
		"status":   map[string]interface{}{"volumeGroupSnapshotName": "group-1", "boundVolumeSnapshotContentName": "member-content"},
	}}
}

func unitMemberContent() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": SnapshotAPIVersion, "kind": "VolumeSnapshotContent", "metadata": map[string]interface{}{"name": "member-content"},
		"spec": map[string]interface{}{"volumeSnapshotRef": map[string]interface{}{"name": unitMember().GetName(), "namespace": vgsUnitNamespace}},
	}}
}

func TestVGSCleanup(t *testing.T) {
	r := vgsResources()
	for _, leave := range []string{"", "group-content", "member", "member-content"} {
		t.Run("remaining="+leave, func(t *testing.T) {
			group, content, member, memberContent := unitVGS(), unitVGSContent(), unitMember(), unitMemberContent()
			client := unitDynamic(group, content, member, memberContent)
			client.PrependReactor("delete", "volumegroupsnapshots", func(action clienttesting.Action) (bool, runtime.Object, error) {
				if err := client.Tracker().Delete(r.groupSnapshot, vgsUnitNamespace, "group-1"); err != nil && !apierrs.IsNotFound(err) {
					return true, nil, err
				}
				if leave != "group-content" {
					_ = client.Tracker().Delete(r.groupSnapshotContent, "", content.GetName())
				}
				if leave != "member" {
					_ = client.Tracker().Delete(r.volumeSnapshot, vgsUnitNamespace, member.GetName())
				}
				if leave != "member-content" {
					_ = client.Tracker().Delete(r.volumeSnapshotContent, "", memberContent.GetName())
				}
				return true, nil, nil
			})
			record := newVGSRecord(client, group, []string{"vol-1"})
			err := record.deleteAndWait(context.Background(), time.Millisecond, 5*time.Millisecond)
			if (err != nil) != (leave != "") {
				t.Fatalf("remaining=%s error=%v", leave, err)
			}
			if !record.contents["group-content"] || !record.snapshotContents["member-content"] {
				t.Fatalf("generated resources were not discovered: %+v", record)
			}
			for _, action := range client.Actions() {
				if d, ok := action.(clienttesting.DeleteAction); ok && action.GetResource() == r.groupSnapshot {
					options := d.GetDeleteOptions()
					if options.PropagationPolicy == nil || *options.PropagationPolicy != metav1.DeletePropagationBackground {
						t.Fatal("group deletion must not deadlock member finalizers")
					}
				}
			}
		})
	}
	t.Run("partial setup discovers content without saved names", func(t *testing.T) {
		client := unitDynamic(unitVGS(), unitVGSContent())
		record := newVGSRecord(client, unitVGS(), nil)
		_, err := record.observe(context.Background())
		if err != nil || !record.contents["group-content"] {
			t.Fatalf("record %v, err %v", record, err)
		}
	})
	t.Run("already deleted is safe", func(t *testing.T) {
		record := newVGSRecord(unitDynamic(), unitVGS(), []string{"vol-1"})
		for i := 0; i < 2; i++ {
			if err := record.deleteAndWait(context.Background(), time.Millisecond, time.Second); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("unrelated resources are not mistaken for leftovers", func(t *testing.T) {
		content := unitVGSContent()
		_ = unstructured.SetNestedField(content.Object, "another-uid", "spec", "volumeGroupSnapshotRef", "uid")
		record := newVGSRecord(unitDynamic(content), unitVGS(), nil)
		if err := record.deleteAndWait(context.Background(), time.Millisecond, time.Second); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("cleanup API failure is reported", func(t *testing.T) {
		client := unitDynamic()
		client.PrependReactor("list", "volumegroupsnapshotcontents", func(clienttesting.Action) (bool, runtime.Object, error) { return true, nil, errors.New("forbidden") })
		if err := newVGSRecord(client, unitVGS(), nil).deleteAndWait(context.Background(), time.Millisecond, time.Second); err == nil {
			t.Fatal("expected cleanup failure")
		}
	})
}

func TestVGSPartialFixtureCleanup(t *testing.T) {
	r := vgsResources()
	class := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": r.apiVersion, "kind": "VolumeGroupSnapshotClass", "metadata": map[string]interface{}{"name": "created-class"}}}
	client := unitDynamic(class)
	f := newVGSFixture(kubefake.NewSimpleClientset(), client, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vgsUnitNamespace}})
	f.classes = []string{class.GetName()}
	if err := f.cleanupResources(context.Background(), time.Millisecond, time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Resource(r.groupSnapshotClass).Get(context.Background(), class.GetName(), metav1.GetOptions{}); !apierrs.IsNotFound(err) {
		t.Fatalf("class remains: %v", err)
	}
}

func TestVGSClaimAndPodDeletionAreRepeatable(t *testing.T) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vgsUnitNamespace}}
	claim := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "claim", Namespace: vgsUnitNamespace}, Spec: v1.PersistentVolumeClaimSpec{VolumeName: "pv"}}
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: vgsUnitNamespace}}
	client := kubefake.NewSimpleClientset(claim, pod)
	pvc := &TestPersistentVolumeClaim{namespace: ns, persistentVolumeClaim: claim}
	tpod := &TestPod{namespace: ns, pod: pod}
	for i := 0; i < 2; i++ {
		if err := deleteVGSPod(context.Background(), client, tpod); err != nil {
			t.Fatal(err)
		}
		if err := deleteVGSClaim(context.Background(), client, pvc); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVGSFixtureCleanupOrder(t *testing.T) {
	for _, failGroupDelete := range []bool{false, true} {
		t.Run(fmt.Sprintf("group_delete_failure=%v", failGroupDelete), func(t *testing.T) {
			r := vgsResources()
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vgsUnitNamespace}}
			source := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: vgsUnitNamespace}}
			restore := &v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: vgsUnitNamespace}}
			pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: vgsUnitNamespace}}
			cs := kubefake.NewSimpleClientset(source, restore, pod)
			class := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": r.apiVersion, "kind": "VolumeGroupSnapshotClass", "metadata": map[string]interface{}{"name": "class"}}}
			dc := unitDynamic(unitVGS(), class)
			var order []string
			cs.PrependReactor("delete", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
				order = append(order, "pod")
				return false, nil, nil
			})
			cs.PrependReactor("delete", "persistentvolumeclaims", func(a clienttesting.Action) (bool, runtime.Object, error) {
				order = append(order, a.(clienttesting.DeleteAction).GetName())
				return false, nil, nil
			})
			dc.PrependReactor("delete", "volumegroupsnapshots", func(clienttesting.Action) (bool, runtime.Object, error) {
				order = append(order, "group")
				if failGroupDelete {
					return true, nil, errors.New("delete denied")
				}
				return false, nil, nil
			})
			dc.PrependReactor("delete", "volumegroupsnapshotclasses", func(clienttesting.Action) (bool, runtime.Object, error) {
				order = append(order, "class")
				return false, nil, nil
			})
			f := newVGSFixture(cs, dc, ns)
			f.claims = []vgsClaim{
				// Simulate a binding helper failure that lost its live PVC object.
				{pvc: &TestPersistentVolumeClaim{namespace: ns}, created: source},
				{pvc: &TestPersistentVolumeClaim{namespace: ns, persistentVolumeClaim: restore}, created: restore, restored: true},
			}
			f.pods = []*TestPod{{namespace: ns, pod: pod}}
			f.groups = []*vgsRecord{newVGSRecord(dc, unitVGS(), nil)}
			f.classes = []string{"class"}
			err := f.cleanupResources(context.Background(), time.Millisecond, 5*time.Millisecond)
			if (err != nil) != failGroupDelete {
				t.Fatalf("cleanup error=%v", err)
			}
			if !reflect.DeepEqual(order, []string{"pod", "restore", "group", "source", "class"}) {
				t.Fatalf("wrong cleanup order: %v", order)
			}
		})
	}
}
