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
	"github.com/IBM/ibmcloud-storage-cbr/cbr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("[ics-e2e] [exec-cvmp] [pods-seq] POD with Common Volumes(PVCs)", func() {
	var (
		cs           clientset.Interface
		cleanupFuncs []func()
		volList      []testsuites.VolumeDetails
		cmdShotLife  string
		//cmdLongLife  string
		//withPVC bool
		maxPVC int
		maxPOD int
		fw     *framework.Framework
		ns     *v1.Namespace
	)

	//withPVC = true

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

	fw = framework.NewDefaultFramework("ics-e2e-cvmp")

	BeforeEach(func() {
		cs = fw.ClientSet
		ns = fw.Namespace
		cleanupFuncs = make([]func(), 0)

		cmdShotLife = "df -h; echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"
		//cmdLongLife = "df -h; echo 'hello world' > /mnt/test-1/data && while true; do sleep 2; done"

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

	It("should create multiple pods in sequence with common PVC(s), write and read to volume", func() {
		By("create multiple pods in sequence with common PVC(s)")
		//Modifying the E2E to show how to use the lib in existing e2e tests cases by applying cbr rules using new library
		var execCmd string
		var cmdExits bool
		var vols []testsuites.VolumeDetails
		var pods []testsuites.PodDetails
		cbrlib := testsuites.GetCBRContext()

		vollistLen := len(volList)
		vols = make([]testsuites.VolumeDetails, 0)
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
		execCmd = cmdShotLife
		cmdExits = true
		for n := range vols {
			_, funcs := vols[n].SetupDynamicPersistentVolumeClaim(cs, ns, false)
			cleanupFuncs = append(cleanupFuncs, funcs...)
		}
		//create cbr
		vpcCRN := os.Getenv("vpcCRN")
		cbrInput := cbr.CBR{
			VPC: []string{vpcCRN},
		}
		//Assert Failure with logs
		zoneID, err := cbrlib.CreateCBRZone("E2E CA Zone", cbrInput)
		Expect(err).NotTo(HaveOccurred())
		Expect(zoneID).ShouldNot(BeNil())

		rule, err := testsuites.CreateZoneRules(cbrlib, "is", zoneID)
		Expect(err).NotTo(HaveOccurred())
		Expect(rule).ShouldNot(BeNil())

		for i := range cleanupFuncs {
			defer cleanupFuncs[i]()
		}

		pods = make([]testsuites.PodDetails, 0)
		for i := 0; i < maxPOD; i++ {
			pod := testsuites.PodDetails{
				Cmd:      execCmd,
				CmdExits: cmdExits,
				Volumes:  vols,
			}
			pods = append(pods, pod)
		}

		test := testsuites.DynamicallyProvisioneMultiPodWithVolTest{
			Pods: pods,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n",
			},
		}
		test.RunSync(cs, ns)
		//Pod creation must fail
		//Delete zone and rule
		err = cbrlib.DeleteCBRRuleZone(rule, zoneID)
		Expect(err).ShouldNot(BeNil())
	})

})

var _ = Describe("[ics-e2e] [exec-cvmp] [pods-simul] POD with Common Volumes(PVCs)", func() {
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

	It("should create multiple pods in parallel with common PVC(s), write and read to volume", func() {
		By("create multiple pods in parallel with common PVC(s)")
		var execCmd string
		var cmdExits bool
		var vols []testsuites.VolumeDetails
		var pods []testsuites.PodDetails

		vollistLen := len(volList)
		vols = make([]testsuites.VolumeDetails, 0)
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

		pods = make([]testsuites.PodDetails, 0)
		for i := 0; i < maxPOD; i++ {
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

var _ = Describe("[ics-e2e] [exec-cvmp] [deploy] Deployment with Common Volumes(PVCs)", func() {
	var (
		cs      clientset.Interface
		fw      *framework.Framework
		ns      *v1.Namespace
		maxPVC  int
		volList []testsuites.VolumeDetails
	)

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
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		accessMode := v1.ReadWriteOnce
		volList = []testsuites.VolumeDetails{
			{
				PVCName:       "ics-vol-5iops-",
				VolumeType:    "ibmc-vpc-block-5iops-tier",
				FSType:        "ext4",
				ClaimSize:     "10Gi",
				ReclaimPolicy: &reclaimPolicy,
				AccessMode:    &accessMode,
				MountOptions:  []string{"rw"},
				VolumeMount: testsuites.VolumeMountDetails{
					NameGenerate:      "test-volume-",
					MountPathGenerate: "/mnt/test-",
				},
			},
			{
				PVCName:       "ics-vol-5iops-",
				VolumeType:    "ibmc-vpc-block-5iops-tier",
				FSType:        "ext4",
				ClaimSize:     "11Gi",
				ReclaimPolicy: &reclaimPolicy,
				AccessMode:    &accessMode,
				MountOptions:  []string{"rw"},
				VolumeMount: testsuites.VolumeMountDetails{
					NameGenerate:      "test-volume-",
					MountPathGenerate: "/mnt/test-",
				},
			},
		}
	})

	It("should create a pvc &pv, deployment resources, write and read to volume, delete the pod, write and read to volume again", func() {
		By("create deployment with PVC(s)")
		var vols []testsuites.VolumeDetails

		vollistLen := len(volList)
		vols = make([]testsuites.VolumeDetails, 0)
		xi := 0
		for i := 0; vollistLen > 0 && i < maxPVC; i++ {
			if xi > vollistLen {
				xi = 0
			}
			vol := volList[xi]
			vols = append(vols, vol)
			xi = xi + 1
		}
		//Create PVC

		pod := testsuites.PodDetails{
			Cmd:      "df -h; echo 'hello world' >> /mnt/test-1/data && while true; do sleep 2; done",
			CmdExits: false,
			Volumes:  vols,
		}

		test := testsuites.DynamicallyProvisioneDeployWithVolWRTest{
			Pod: pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:              []string{"cat", "/mnt/test-1/data"},
				ExpectedString01: "hello world\n",
				ExpectedString02: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.RunMultiVol(cs, ns)
	})

})
