/**
 * Copyright 2020 IBM Corp.
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

// Package utils ...
package utils

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"time"

	grpcClient "github.com/IBM/ibm-csi-common/pkg/utils/grpc-client"
	pb "github.com/IBM/ibm-csi-common/provider"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	endpoint = flag.String("endpoint", "/tmp/storage-secret-sidecar.sock", "Storage secret sidecar endpoint")
)

func UnixConnect(addr string, t time.Duration) (net.Conn, error) {
	unix_addr, err := net.ResolveUnixAddr("unix", addr)
	conn, err := net.DialUnix("unix", nil, unix_addr)
	return conn, err
}

//ClusterInfo contians the cluster information
type ClusterInfo struct {
	ClusterID   string `json:"cluster_id"`
	ClusterName string `json:"cluster_name,omitempty"`
	DataCenter  string `json:"datacenter,omitempty"`
	CustomerID  string `json:"customer_id,omitempty"`
}

//NewClusterInfo loads cluster info
func NewClusterInfo(logger *zap.Logger) (*ClusterInfo, error) {
	configBasePath := config.GetConfPathDir()
	clusterInfo := &ClusterInfo{}
	clusterInfoFile := filepath.Join(configBasePath, ClusterInfoPath)
	clusterInfoContent, err := ioutil.ReadFile(filepath.Clean(clusterInfoFile))
	if err != nil {
		logger.Error("Error while reading  cluster-config.json", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal([]byte(clusterInfoContent), clusterInfo)
	if err != nil {
		logger.Error("Error while parsing cluster-config", zap.Error(err))
		return nil, err
	}
	return clusterInfo, nil

}

// APIKeyImpl implementation
type APIKeyImpl struct {
	logger *zap.Logger
}

//NewAPIKey returns the new decryptor
func NewAPIKey(loggerIn *zap.Logger) (*APIKeyImpl, error) {
	var err error
	apiKeyImp := &APIKeyImpl{
		logger: loggerIn,
	}
	return apiKeyImp, err
}

//UpdateIAMKeys decrypts the API keys and updates.
func (d *APIKeyImpl) UpdateIAMKeys(config *config.Config) error {
	//Setup grpc connection
	var GRPCBackend grpcClient.GrpcSessionFactory
	grpcSess := GRPCBackend.NewGrpcSession()
	cc := &grpcClient.GrpcSes{}
	conn, err := grpcSess.GrpcDial(cc, *endpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDialer(UnixConnect))
	if err != nil {
		err = fmt.Errorf("failed to establish grpc-client connection: %v", err)
		return err
	}

	//APIKeyProvider Client
	c := pb.NewAPIKeyProviderClient(conn)
	_, requestID := GetContextLogger(context.Background(), false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if config.Bluemix.Encryption {
		r, err := c.GetContainerAPIKey(ctx, &pb.Cipher{Cipher: config.Bluemix.IamAPIKey, RequestID: requestID})
		if err != nil {
			return err
		}
		config.Bluemix.IamAPIKey = r.GetApikey()
	}
	if config.VPC.Encryption {
		if config.VPC.APIKey != "" {
			r, err := c.GetVPCAPIKey(ctx, &pb.Cipher{Cipher: config.VPC.APIKey, RequestID: requestID})
			if err != nil {
				return err
			}
			config.VPC.APIKey = r.GetApikey()
		}
		if config.VPC.G2APIKey != "" {
			r, err := c.GetVPCAPIKey(ctx, &pb.Cipher{Cipher: config.VPC.G2APIKey, RequestID: requestID})
			if err != nil {
				return err
			}
			config.VPC.G2APIKey = r.GetApikey()
		}
	}
	return nil
}
