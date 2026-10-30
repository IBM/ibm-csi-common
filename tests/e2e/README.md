## How to execute E2E?

1. Create a VPC Cluster
2. Export the KUBECONFIG
   In kube config file use abosulte path for `certificate-authority`, `client-certificate` and `client-key`
3. Deploy the Driver (with SC)
4. Export enviornment variables
   ```
   # Mandatory
   export GO111MODULE=on
   export GOPATH=<GOPATH>
   export E2E_TEST_RESULT=<absolute-path to a file where the results should be redirected>
   export TEST_ENV=<stage/prod>
   export IC_REGION=<us-south>
   export IC_API_KEY_PROD=<prod API key> | export IC_API_KEY_STAG=<stage API key>
   export icrImage=<Give the image which will be used by pods>
   
   # Optional
   export E2E_POD_COUNT="1"
   export E2E_PVC_COUNT="1"
   ```

5. Test all SC with deployment
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[sc\] \[with-deploy\]"  ./tests/e2e
   ```
6. Test all SC with pod
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[sc\] \[with-pods\]"  ./tests/e2e
   ```
7. Test 5 IOPS SC with statefulset(with 2 replicas)
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[sc\] \[with-statefulset\]"  ./tests/e2e
   ```
8. Test multiple volumes with deployment
   ```
   export E2E_PVC_COUNT="2"
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[exec-cvmp\] \[deploy\]" ./tests/e2e
   ```
9. Test multiple volumes with multiple pods. In following example, two PVC will be created and four pods will be created in sequence using same two PVCs
   ```
   export E2E_PVC_COUNT="2"
   export E2E_POD_COUNT="4"
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[exec-cvmp\] \[pods-seq\]" ./tests/e2e
   ```
10. Test concurrent pods deployment with two PVC each
   ```
   export E2E_PVC_COUNT="2"
   ginkgo -v -nodes=5 --focus="\[ics-e2e\] \[exec-mvmp\] \[pods-conc\]" ./tests/e2e
   ```
11. Run all SC test in parallel
   ```
   ginkgo -v -nodes=4 --focus="\[ics-e2e\] \[sc\]"  ./tests/e2e
   ```
12. Test node drain scenario
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[node-drain\] \[with-pods\]" ./tests/e2e
   ```
13. Test acadia profile and it's storage classes
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[with-sdp-profile\]"  ./tests/e2e
   ```
14. Test volume group snapshot creation, restore, data integrity, and deletion.
    These tests require an existing cluster serving `groupsnapshot.storage.k8s.io/v1`
    and `snapshot.storage.k8s.io/v1`, working VGS controllers/sidecars, and
    IBM VPC Block CSI Driver version **5.2 only**. The runner does not install
    CRDs or enable VGS. The E2E account needs access to group snapshots, classes,
    contents, member snapshots/contents, pods, PVCs, PVs, events, and volume attachments.
    The `ibmc-vpc-block-5iops-tier` storage class must use Immediate binding and
    Delete reclaim policy. Supply a writable shell-capable pod image via `icrImage`.
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[vgs\]" ./tests/e2e -- -e2e-verify-service-account=false
   ```

   In Jenkins automation, enable the optional suite by passing the checkbox value
   to the E2E runner:
   ```
   ./tests/e2e/e2e.sh --login --env <stage|prod> --region <region> --run-vgs true
   ```

   The E2E runner executes this suite only when the normalized VPC Block CSI
   add-on version is exactly `5.2`.

   The nine independent scenarios cover the basic lifecycle, exclusion of an
   unlabeled third PVC, point-in-time data, recovery after source pod/PVC/PV
   deletion, deletion during creation, repeated Kubernetes deletion, and negative
   cases for no matching PVCs, unattached source volumes, and a missing class.
   The repeated-delete case checks Kubernetes API behavior, not CSI RPC idempotency.
   Negative cases must report the expected cause; an unrelated error or timeout
   does not count as success. Delete-during-creation is skipped if creation finishes
   before the test can observe bound, unready content; rerun that case to exercise it.

   Each scenario records PASS only after cleanup. Failures include VGS status and
   namespace events. Cleanup checks group/member snapshots and their contents,
   including resources created during partial setup. Unit tests can run without a cluster:
   ```
   go test ./tests/e2e/testsuites
   ```
