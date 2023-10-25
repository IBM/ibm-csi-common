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

5. Test DP2 profile with deployment
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[sc\] \[with-deploy\]"  ./tests/e2e_file
   ```
6. Test volume expansion
   ```
   ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[resize\] \[pv\]"  ./tests/e2e_file
   ```