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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIBMCloudStorageProvider(t *testing.T) {
	// Creating test logger
	logger, teardown := GetTestLogger(t)
	defer teardown()

	pwd, err := os.Getwd()
	if err != nil {
		t.Errorf("Failed to get current working directory, some unit tests will fail")
	}

	// As its required by NewIBMCloudStorageProvider
	secretConfigPath := filepath.Join(pwd, "..", "..", "test-fixtures", "valid")
	err = os.Setenv("SECRET_CONFIG_PATH", secretConfigPath)
	defer os.Unsetenv("SECRET_CONFIG_PATH")
	if err != nil {
		t.Errorf("This test will fail because of %v", err)
	}

	configPath := filepath.Join(pwd, "..", "..", "test-fixtures", "slconfig.toml")
	ibmCloudProvider, err := NewIBMCloudStorageProvider(configPath, logger)
	assert.NotNil(t, err)
	assert.Nil(t, ibmCloudProvider)
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

	clusterInfo := ibmFakeCloudProvider.GetClusterInfo()
	assert.NotNil(t, clusterInfo)
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
