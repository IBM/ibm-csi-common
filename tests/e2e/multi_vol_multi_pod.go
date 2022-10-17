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
	"os"
	"strconv"

	"github.com/IBM/ibm-csi-common/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("[ics-e2e] [exec-mvmp] [pods-simul] Multiple POD with PVCs", func() {
	var (
		cs           clientset.Interface
		cleanupFuncs []func()
		volList      []testsuites.VolumeDetails
		//cmdShotLife  string
		cmdLongLife string
		withPVC     bool
		maxPVC      int
		maxPOD      int
		fw          *framework.Framework
		ns          *v1.Namespace
	)

	withPVC = true
	envOpt := os.Getenv("E2E_NOPVC")
	if envOpt == "yes" {
		withPVC = false
	}

	maxPOD = 1
	podCount := os.Getenv("E2E_POD_COUNT")
	if len(podCount) > 0 {
		var err error
		maxPOD, err = strconv.Atoi(podCount)
		if err != nil {
			maxPOD = 1
		}
	}

	maxPVC = 1
	pvcCount := os.Getenv("E2E_PVC_COUNT")
	if len(pvcCount) > 0 {
		var err error
		maxPVC, err = strconv.Atoi(pvcCount)
		if err != nil {
			maxPVC = 1
		}
	}

	fw = framework.NewDefaultFramework("ics-e2e-mvmp")

	BeforeEach(func() {
		cs = fw.ClientSet
		ns = fw.Namespace
		cleanupFuncs = make([]func(), 0)

		//cmdShotLife = "df -h; echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"
		cmdLongLife = "df -h; echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done"

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		accessMode := v1.ReadWriteOnce

		volList = []testsuites.VolumeDetails{
			{
				PVCName:       "ics-vol-5iops-",
				VolumeType:    "ibmc-vpc-block-5iops-tier",
				FSType:        "ext4",
				ClaimSize:     "11Gi",
				ReclaimPolicy: &reclaimPolicy,
				MountOptions:  []string{"rw"},
				AccessMode:    &accessMode,
				VolumeMount: testsuites.VolumeMountDetails{
					NameGenerate:      "test-volume-",
					MountPathGenerate: "/mnt/test-",
					ReadOnly:          false,
				},
			},
			{
				PVCName:       "ics-vol-gp-",
				VolumeType:    "ibmc-vpc-block-general-purpose",
				FSType:        "ext4",
				ClaimSize:     "12Gi",
				ReclaimPolicy: &reclaimPolicy,
				MountOptions:  []string{"rw"},
				AccessMode:    &accessMode,
				VolumeMount: testsuites.VolumeMountDetails{
					NameGenerate:      "test-volume-",
					MountPathGenerate: "/mnt/test-",
					ReadOnly:          false,
				},
			},
			{
				PVCName:       "ics-vol-5iops-",
				VolumeType:    "ibmc-vpc-block-5iops-tier",
				FSType:        "ext4",
				ClaimSize:     "13Gi",
				ReclaimPolicy: &reclaimPolicy,
				MountOptions:  []string{"rw"},
				AccessMode:    &accessMode,
				VolumeMount: testsuites.VolumeMountDetails{
					NameGenerate:      "test-volume-",
					MountPathGenerate: "/mnt/test-",
					ReadOnly:          false,
				},
			},
		}
	})

	It("should create multiple pods in parallel with PVC(s), write and read to volume", func() {
		By("create multiple pods in parallel with PVC(s)")
		var execCmd string
		var cmdExits bool
		var pods []testsuites.PodDetails

		vollistLen := len(volList)

		pods = make([]testsuites.PodDetails, 0)
		for i := 0; i < maxPOD; i++ {
			framework.Logf("creating PVC(s) for POD %d", i+1)

			vols := make([]testsuites.VolumeDetails, 0)
			xi := 0
			for i := 0; vollistLen > 0 && i < maxPVC; i++ {
				if xi >= vollistLen {
					xi = 0
				}
				vol := volList[xi]
				vols = append(vols, vol)
				xi = xi + 1
			}

			//Create PVC
			if withPVC {
				execCmd = cmdLongLife
				cmdExits = false
				for n := range vols {
					_, funcs := vols[n].SetupDynamicPersistentVolumeClaim(cs, ns, false)
					cleanupFuncs = append(cleanupFuncs, funcs...)
				}
			} else {
				// Without PVC
				execCmd = "echo 'hello world' && while true; do sleep 2; done"
				cmdExits = false
			}

			for i := range cleanupFuncs {
				defer cleanupFuncs[i]()
			}

			pod := testsuites.PodDetails{
				Cmd:      execCmd,
				CmdExits: cmdExits,
				Volumes:  vols,
			}
			pods = append(pods, pod)
		}

		test := testsuites.DynamicallyProvisioneMultiPodWithVolTest{
			Pods:     pods,
			PodCheck: nil,
		}
		test.RunAsync(cs, ns)
	})
})
