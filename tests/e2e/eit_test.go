/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2024 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeClient *kubernetes.Clientset
var namespace string = "kube-system" // Change this to your namespace

const (
	workerPoolLabelKey = "ibm-cloud.kubernetes.io/worker-pool-name"
	hostnamekey        = "kubernetes.io/hostname"
)

func TestConfigMap(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "ConfigMap Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	kubeConfigPath := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	kubeClient, err = kubernetes.NewForConfig(config)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})

var _ = ginkgo.Describe("Enable EIT on all worker pools", func() {
	const (
		configMapName     = "addon-vpc-file-csi-driver-configmap"
		validationMapName = "file-csi-driver-status"
	)

	ginkgo.It("Enable EIT on all worker pools", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = ""
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})

	ginkgo.It("Enable EIT on only one worker pool", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})

	ginkgo.It("enable EIT on multiple worker pools", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default", "worker-pool1"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default,worker-pool1"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})

	ginkgo.It("Disable EIT on all worker pools", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "false"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		hostnames := make(map[string][]string)
		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}
		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})

	ginkgo.It("Enable EIT on one worker pool, check the updated worker pool list, update with one more worker pool and verify", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))

		// Retrieve the existing ConfigMap
		configMap, err = kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		// Adding worker pool
		workerpools = []string{"default", "worker-pool1"}
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default,worker-pool1"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err = labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector = labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err = kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames = make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err = yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})

	ginkgo.It("Enable EIT on a non existing worker pool", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "non-exist"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})

	ginkgo.It("Enable EIT on multiple worker pools, remove one worker pool and verify", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default", "worker-pool1"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default,worker-pool1"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))

		// Retrieve the existing ConfigMap
		configMap, err = kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		// Removing one worker pool
		workerpools = []string{"default"}
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err = labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector = labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err = kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames = make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err = yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})

})

/*
var _ = ginkgo.Describe("Enable EIT on only one worker pool", func() {
	const (
		configMapName     = "addon-vpc-file-csi-driver-configmap"
		validationMapName = "file-csi-driver-status"
	)

	ginkgo.It("should update the feature flag config map and check for events in status configmap", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})
})
*/

/*
var _ = ginkgo.Describe("Enable EIT on only multiple worker pool", func() {
	const (
		configMapName     = "addon-vpc-file-csi-driver-configmap"
		validationMapName = "file-csi-driver-status"
	)

	ginkgo.It("should update the feature flag config map and check for events in status configmap", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default", "worker-pool1"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default,worker-pool1"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})
})
*/

/*
var _ = ginkgo.Describe("Disable EIT on all worker pools", func() {
	const (
		configMapName     = "addon-vpc-file-csi-driver-configmap"
		validationMapName = "file-csi-driver-status"
	)

	ginkgo.It("should update the feature flag config map and check for events in status configmap", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "false"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		hostnames := make(map[string][]string)
		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}
		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})
})
*/

/*

var _ = ginkgo.Describe("Enable EIT on one worker pool, check the updated worker pool list, update with one more worker pool and verify", func() {
	const (
		configMapName     = "addon-vpc-file-csi-driver-configmap"
		validationMapName = "file-csi-driver-status"
	)

	ginkgo.It("should update the feature flag config map and check for events in status configmap", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))

		// Adding worker pool
		workerpools = []string{"default", "worker-pool1"}
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default,worker-pool1"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err = labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector = labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err = kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames = make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err = yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})
})
*/

/*
var _ = ginkgo.Describe("Enable EIT on a non existing worker pool", func() {
	const (
		configMapName     = "addon-vpc-file-csi-driver-configmap"
		validationMapName = "file-csi-driver-status"
	)

	ginkgo.It("should update the feature flag config map and check for events in status configmap", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "non-exist"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})
})
*/

/*
var _ = ginkgo.Describe("Enable EIT on multiple worker pools, remove one worker pool and verify", func() {
	const (
		configMapName     = "addon-vpc-file-csi-driver-configmap"
		validationMapName = "file-csi-driver-status"
	)

	ginkgo.It("should update the feature flag config map and check for events in status configmap", func() {
		ctx := context.Background()

		// Retrieve the existing ConfigMap
		configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerpools := []string{"default", "worker-pool1"}
		// Update a key in the ConfigMap
		configMap.Data["ENABLE_EIT"] = "true"
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default,worker-pool1"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err := labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector := labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames := make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err := yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))

		// Removing one worker pool
		workerpools = []string{"default"}
		configMap.Data["EIT_ENABLED_WORKER_POOLS"] = "default"
		_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		workerPoolReq, err = labels.NewRequirement(workerPoolLabelKey, selection.In, workerpools)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		selector = labels.NewSelector()
		selector = selector.Add(*workerPoolReq)
		nodeList, err = kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		hostnames = make(map[string][]string)
		for _, node := range nodeList.Items {
			hosts, ok := hostnames[node.Labels[workerPoolLabelKey]]
			if ok {
				hostnames[node.Labels[workerPoolLabelKey]] = append(hosts, node.Name)
				continue
			}
			hostnames[node.Labels[workerPoolLabelKey]] = []string{node.Name}
		}

		workerNodeYaml, err = yaml.Marshal(hostnames)
		if err != nil {
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		gomega.Eventually(func() (map[string]string, error) {
			validationConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, validationMapName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return validationConfigMap.Data, nil
		}, 5*time.Minute, 60*time.Second).Should(gomega.HaveKeyWithValue("EIT_ENABLED_WORKER_NODES", string(workerNodeYaml)))
		// Give some time for the changes to propagate
	})
})
*/
