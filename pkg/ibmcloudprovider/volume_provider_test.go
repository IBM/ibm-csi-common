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

// Package ibmcloudprovider ...
package ibmcloudprovider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	sp "github.com/IBM/secret-utils-lib/pkg/secret_provider"
	"github.com/stretchr/testify/assert"
)

func TestNewIBMCloudStorageProvider(t *testing.T) {
	// Creating test logger
	logger, teardown := GetTestLogger(t)
	defer teardown()

	testcases := []struct {
		testcasename    string
		clusterConfPath string
		secretConfPath  string
		iksEnabled      string
		expectedError   error
	}{
		{
			testcasename:    "Successful initialization of cloud storage provider",
			clusterConfPath: "test-fixtures/valid/cluster_info/cluster-config.json",
			secretConfPath:  "test-fixtures/slconfig.toml",
			iksEnabled:      "True",
			expectedError:   nil,
		},
		{
			testcasename:    "Invalid cluster config",
			clusterConfPath: "test-fixtures/invalid/cluster_info/non-exist.json",
			secretConfPath:  "test-fixtures/slconfig.toml",
			iksEnabled:      "True",
			expectedError:   errors.New("not nil"),
		},
		{
			testcasename:    "Secret config doesn't exist",
			clusterConfPath: "test-fixtures/valid/cluster_info/cluster-config.json",
			secretConfPath:  "test-fixtures/non-exist.toml",
			iksEnabled:      "True",
			expectedError:   errors.New("not nil"),
		},
		{
			testcasename:    "Both IKS and VPC disabled",
			clusterConfPath: "test-fixtures/valid/cluster_info/cluster-config.json",
			secretConfPath:  "test-fixtures/provider-disabled.toml",
			iksEnabled:      "False",
			expectedError:   errors.New("not nil"),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.testcasename, func(t *testing.T) {
			kc, _ := k8s_utils.FakeGetk8sClientSet(logger)
			pwd, err := os.Getwd()
			if err != nil {
				t.Errorf("Failed to get current working directory, test related to read config will fail, error: %v", err)
			}

			clusterConfPath := filepath.Join(pwd, "..", "..", testcase.clusterConfPath)
			_ = k8s_utils.FakeCreateCM(kc, clusterConfPath)

			secretConfPath := filepath.Join(pwd, "..", "..", testcase.secretConfPath)
			_ = k8s_utils.FakeCreateSecret(kc, "DEFAULT", secretConfPath)

			os.Setenv("IKS_ENABLED", testcase.iksEnabled)
			spObject := new(sp.FakeSecretProvider)
			_, err = NewIBMCloudStorageProvider("test", kc, spObject, logger)
			if testcase.expectedError != nil {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestNewFakeIBMCloudStorageProvider(t *testing.T) {
	// Creating test logger
	logger, teardown := GetTestLogger(t)
	defer teardown()

	pwd, err := os.Getwd()
	if err != nil {
		t.Errorf("Failed to get current working directory, some unit tests will fail")
	}

	// As its required by NewFakeIBMCloudStorageProvider
	secretConfigPath := filepath.Join(pwd, "..", "..", "test-fixtures", "valid")
	err = os.Setenv("SECRET_CONFIG_PATH", secretConfigPath)
	defer os.Unsetenv("SECRET_CONFIG_PATH")
	if err != nil {
		t.Errorf("This test will fail because of %v", err)
	}

	configPath := filepath.Join(pwd, "..", "..", "test-fixtures", "slconfig.toml")
	ibmFakeCloudProvider, err := NewFakeIBMCloudStorageProvider(configPath, logger)
	assert.Nil(t, err)
	assert.NotNil(t, ibmFakeCloudProvider)

	fakeSession, err := ibmFakeCloudProvider.GetProviderSession(context.TODO(), logger)
	assert.Nil(t, err)
	assert.NotNil(t, fakeSession)

	cloudProviderConfig := ibmFakeCloudProvider.GetConfig()
	assert.NotNil(t, cloudProviderConfig)

	clusterID := ibmFakeCloudProvider.GetClusterID()
	assert.NotNil(t, clusterID)
}

func TestCorrectEndpointURL(t *testing.T) {
	testCases := []struct {
		name      string
		url       string
		returnURL string
	}{
		{
			name:      "URL of http form",
			url:       "http://example.com",
			returnURL: "https://example.com",
		},
		{
			name:      "URL of https form",
			url:       "https://example.com",
			returnURL: "https://example.com",
		},
		{
			name:      "Incorrect URL",
			url:       "xyz.com",
			returnURL: "xyz.com",
		},
		{
			name:      "Incorrect URL",
			url:       "httpd://xyz.com",
			returnURL: "httpd://xyz.com",
		},
	}
	logger, teardown := GetTestLogger(t)
	defer teardown()

	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		url := getEndpointURL(tc.url, logger)
		assert.Equal(t, tc.returnURL, url)
	}
}
