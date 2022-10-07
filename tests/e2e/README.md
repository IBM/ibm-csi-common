## How to execute E2E?

1. Create a VPC Cluster
2. Export the KUBECONFIG
   In kube config file use abosulte path for `certificate-authority`, `client-certificate` and `client-key`
3. Deploy the Driver (with SC)
4. Export enviornment variables
   ```
   export E2E_POD_COUNT="1"
   export E2E_PVC_COUNT="1"
   export GO111MODULE=on
   export GOPATH=<GOPATH>
   export E2E_TEST_RESULT=<absolute-path to a file where the results should be redirected>
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

## How to execute E2E for custom deployments for example we want to test for different sidecare or driver images?

1. Upload the deployment files in format addonVersion_platform.yaml. For e.g. for testing 5.0 addon on openshift platform it will be 5.0_openshift.yaml

2. Go to Jenkins job https://alchemy-containers-jenkins.swg-devops.com/job/Containers-Volumes/job/ibm-vpc-csi-block-e2e-custom-dep-2022/ and run job for different addon versions and cluster versions.
