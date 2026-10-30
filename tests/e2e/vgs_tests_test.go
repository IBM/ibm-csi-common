package e2e

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	apps "k8s.io/api/apps/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestVGSResultCatalog(t *testing.T) {
	data, err := os.ReadFile("conf/testcases.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog map[string][]string
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	entries := map[string]bool{}
	for _, key := range catalog["vgs"] {
		entries[key] = true
	}
	tags := []string{"[positive] basic", "[selector-isolation]", "[point-in-time]", "[restore-after-source-delete]", "[delete-while-creating]", "[delete-idempotency]", "[no-pvcs]", "[unattached]", "[invalid-class]"}
	for _, tag := range tags {
		key := vgsResultKey(tag)
		if !entries[key] {
			t.Fatalf("missing or duplicate result key %q", key)
		}
		delete(entries, key)
	}
	if len(entries) != 0 {
		t.Fatalf("unused result entries: %v", entries)
	}
}

func TestVGSDriverVersion(t *testing.T) {
	for _, tc := range []struct {
		version, want string
		invalid       bool
	}{
		{"v5.2.14_iks", "5.2", false}, {"5.2", "5.2", false}, {"v5.1.47_ocp", "5.1", false},
		{"5.3.0", "5.3", false}, {"5.20.0", "5.20", false}, {"", "", true}, {"5", "", true}, {"5.2x.1", "", true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			cs := kubefake.NewSimpleClientset(&apps.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ibm-vpc-block-csi-controller", Namespace: "kube-system", Annotations: map[string]string{"version": tc.version}}})
			got, err := vpcBlockCSIDriverMajorMinorVersion(cs)
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("got %q, error %v; expected %q invalid=%v", got, err, tc.want, tc.invalid)
			}
		})
	}
	t.Run("StatefulSet fallback", func(t *testing.T) {
		cs := kubefake.NewSimpleClientset(&apps.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "ibm-vpc-block-csi-controller", Namespace: "kube-system", Annotations: map[string]string{"version": "v5.2.3"}}})
		got, err := vpcBlockCSIDriverMajorMinorVersion(cs)
		if err != nil || got != vgsDriverVersion {
			t.Fatalf("version %q, err %v", got, err)
		}
	})
	t.Run("missing driver fails", func(t *testing.T) {
		if _, err := vpcBlockCSIDriverMajorMinorVersion(kubefake.NewSimpleClientset()); err == nil {
			t.Fatal("expected missing driver failure")
		}
	})
	t.Run("lookup permission failure is reported", func(t *testing.T) {
		cs := kubefake.NewSimpleClientset()
		cs.PrependReactor("get", "deployments", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrs.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, "driver", context.Canceled)
		})
		if _, err := vpcBlockCSIDriverMajorMinorVersion(cs); err == nil {
			t.Fatal("expected permission failure")
		}
	})
}
