//go:build linux
// +build linux

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

// Package mountmanager ...
package mountmanager

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	mount "k8s.io/mount-utils"
)

const (
	//socket path
	defaultSocketPath = "/tmp/mysocket.sock"
	// url
	urlPath = "http://unix/api/mount"
)

// MountEITBasedFileShare mounts EIT based FileShare on host system
func (m *NodeMounter) MountEITBasedFileShare(ctxLogger *zap.Logger, stagingTargetPath string, targetPath string, fsType string, requestID string) error {
	// Create payload
	payload := fmt.Sprintf(`{"stagingTargetPath":"%s","targetPath":"%s","fsType":"%s","requestID":"%s"}`, stagingTargetPath, targetPath, fsType, requestID)

	// Get socket path
	socketPath := os.Getenv("SOCKET_PATH")
	if socketPath == "" {
		socketPath = defaultSocketPath
	}
	// Create a custom dialer function for Unix socket connection
	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return net.Dial("unix", socketPath)
	}

	// Create an HTTP client with the Unix socket transport
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: dialer,
		},
	}

	//Create POST request
	req, err := http.NewRequest("POST", urlPath, strings.NewReader(payload))
	if err != nil {
		ctxLogger.Error("Failed to create EIT based request. Failed wth error.")
		return fmt.Errorf("Failed to create EIT based request. Failed wth error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		ctxLogger.Error("Failed to send EIT based request. Failed with error.")
		//TODO: Add retry logic to continuously send request with 5 sec delay. Is it really required?
		// Can we make a systemctl call from here to the local system?
		return fmt.Errorf("Failed to send EIT based request. Failed with error: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		ctxLogger.Error("Error reading response from server")
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Response from server %s.\nResponseCode: %v", string(body), response.StatusCode)
	}

	ctxLogger.Info("Mount passed.", zap.String("Response:", string(body)), zap.Any("StatusCode:", response.StatusCode))
	return nil
}

// MakeFile creates an empty file.
func (m *NodeMounter) MakeFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE, os.FileMode(0644))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}

// MakeDir creates a new directory.
func (m *NodeMounter) MakeDir(path string) error {
	err := os.MkdirAll(path, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// PathExists returns true if the specified path exists.
func (m *NodeMounter) PathExists(path string) (bool, error) {
	return mount.PathExists(path)
}

// GetSafeFormatAndMount returns the existing SafeFormatAndMount object of NodeMounter.
func (m *NodeMounter) GetSafeFormatAndMount() *mount.SafeFormatAndMount {
	return m.SafeFormatAndMount
}

// Resize returns boolean and error if any
func (m *NodeMounter) Resize(devicePath string, deviceMountPath string) (bool, error) {
	r := mount.NewResizeFs(m.GetSafeFormatAndMount().Exec)
	needResize, err := r.NeedResize(devicePath, deviceMountPath)
	if err != nil {
		return false, err
	}
	if needResize {
		if _, err := r.Resize(devicePath, deviceMountPath); err != nil {
			return false, err
		}
	}
	return true, nil
}
