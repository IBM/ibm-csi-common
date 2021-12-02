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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	provider_util "github.com/IBM/ibmcloud-volume-vpc/block/utils"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	"github.com/IBM/ibmcloud-volume-vpc/common/registry"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// IBMCloudStorageProvider Provider
type IBMCloudStorageProvider struct {
	ProviderName   string
	ProviderConfig *config.Config
	Registry       registry.Providers
	ClusterInfo    *utils.ClusterInfo
}

var _ CloudProviderInterface = &IBMCloudStorageProvider{}

// NewIBMCloudStorageProvider ...
func NewIBMCloudStorageProvider(configPath string, logger *zap.Logger) (*IBMCloudStorageProvider, error) {
	logger.Info("NewIBMCloudStorageProvider-Reading provider configuration...")
	// Load config file
	conf, err := config.ReadConfig(configPath, logger)
	if err != nil {
		logger.Fatal("Error loading configuration")
		return nil, err
	}
	// Get only VPC_API_VERSION, in "2019-07-02T00:00:00.000Z" case vpc need only 2019-07-02"
	dateTime, err := time.Parse(time.RFC3339, conf.VPC.APIVersion)
	if err == nil {
		conf.VPC.APIVersion = fmt.Sprintf("%d-%02d-%02d", dateTime.Year(), dateTime.Month(), dateTime.Day())
	} else {
		logger.Warn("Failed to parse VPC_API_VERSION, setting default value")
		conf.VPC.APIVersion = "2020-07-02" // setting default values
	}

	var clusterInfo = &utils.ClusterInfo{}
	logger.Info("Fetching clusterInfo")
	if conf.IKS != nil && conf.IKS.Enabled || os.Getenv("IKS_ENABLED") == "True" {
		clusterInfo, err = utils.NewClusterInfo(logger)
		if err != nil {
			logger.Fatal("Unable to load ClusterInfo", local.ZapError(err))
			return nil, err
		}
		logger.Info("Fetched clusterInfo..")
		if conf.Bluemix.Encryption || conf.VPC.Encryption {
			// api Key if encryption is enabled
			logger.Info("Creating NewAPIKeyImpl...")
			apiKeyImp, err := utils.NewAPIKeyImpl(logger)
			if err != nil {
				logger.Fatal("Unable to create API key getter", local.ZapError(err))
				return nil, err
			}
			logger.Info("Created NewAPIKeyImpl...")
			err = apiKeyImp.UpdateIAMKeys(conf)
			if err != nil {
				logger.Fatal("Unable to get API key", local.ZapError(err))
				return nil, err
			}
		}
	}

	// Update the CSRF  Token
	if conf.Bluemix.PrivateAPIRoute != "" {
		conf.Bluemix.CSRFToken = string([]byte{}) // TODO~ Need to remove it
	}

	if conf.API == nil {
		conf.API = &config.APIConfig{
			PassthroughSecret: string([]byte{}), // // TODO~ Need to remove it
		}
	}
	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    conf.VPC,
		IKSConfig:    conf.IKS,
		APIConfig:    conf.API,
		ServerConfig: conf.Server,
	}
	// Prepare provider registry
	registry, err := provider_util.InitProviders(vpcBlockConfig, logger)
	if err != nil {
		logger.Fatal("Error configuring providers", local.ZapError(err))
	}

	var providerName string
	if isRunningInIKS() && conf.IKS.Enabled {
		providerName = conf.IKS.IKSBlockProviderName
	} else if conf.VPC.Enabled {
		providerName = conf.VPC.VPCBlockProviderName
	}

	cloudProvider := &IBMCloudStorageProvider{
		ProviderName:   providerName,
		ProviderConfig: conf,
		Registry:       registry,
		ClusterInfo:    clusterInfo,
	}
	logger.Info("Successfully read provider configuration")
	return cloudProvider, nil
}

func isRunningInIKS() bool {
	return true //TODO Check the master KUBE version
}

// GetProviderSession ...
func (icp *IBMCloudStorageProvider) GetProviderSession(ctx context.Context, logger *zap.Logger) (provider.Session, error) {
	logger.Info("IBMCloudStorageProvider-GetProviderSession...")
	if icp.ProviderConfig.API == nil {
		icp.ProviderConfig.API = &config.APIConfig{
			PassthroughSecret: string([]byte{}), // // TODO~ Need to remove it
		}
	}

	prov, err := icp.Registry.Get(icp.ProviderName)
	if err != nil {
		logger.Error("Not able to get the said provider, might be its not registered", local.ZapError(err))
		return nil, err
	}

	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    icp.ProviderConfig.VPC,
		IKSConfig:    icp.ProviderConfig.IKS,
		APIConfig:    icp.ProviderConfig.API,
		ServerConfig: icp.ProviderConfig.Server,
	}

	for retryCount := 0; retryCount < utils.MaxRetryAttemptForSessions; retryCount++ {
		session, _, err := provider_util.OpenProviderSessionWithContext(ctx, prov, vpcBlockConfig, icp.ProviderName, logger)
		if err == nil {
			logger.Info("Successfully got the provider session", zap.Reflect("ProviderName", session.ProviderName()))
			return session, nil
		}
		logger.Error("Failed to get provider session", zap.Reflect("Error", err))
		if retryCount == 1 {
			return nil, err
		}
		if providerError, ok := err.(provider.Error); ok && string(providerError.Code()) == provider.APIKeyNotFound {
			// Waiting for minute expecting the API key to be updated in config
			time.Sleep(time.Minute * 1)
			err := icp.UpdateAPIKey(logger)
			if err != nil {
				logger.Error("Failed to update api key in cloud storage provider", zap.Error(err))
				return nil, err
			}
			vpcBlockConfig.VPCConfig.APIKey = icp.ProviderConfig.VPC.G2APIKey
			vpcBlockConfig.VPCConfig.G2APIKey = icp.ProviderConfig.VPC.G2APIKey
			continue
		}
		return nil, err
	}
	return nil, errors.New("API key not found, if it is reset, retry after few minutes")
}

// GetConfig ...
func (icp *IBMCloudStorageProvider) GetConfig() *config.Config {
	return icp.ProviderConfig
}

// GetClusterInfo ...
func (icp *IBMCloudStorageProvider) GetClusterInfo() *utils.ClusterInfo {
	return icp.ClusterInfo
}

// UpdateAPIKey ...
func (icp *IBMCloudStorageProvider) UpdateAPIKey(logger *zap.Logger) error {
	logger.Info("Updating API key in cloud storage provider")
	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    icp.ProviderConfig.VPC,
		IKSConfig:    icp.ProviderConfig.IKS,
		APIConfig:    icp.ProviderConfig.API,
		ServerConfig: icp.ProviderConfig.Server,
	}
	vpcAPIKey := vpcBlockConfig.VPCConfig.G2APIKey

	if icp.ProviderConfig.IKS != nil && (icp.ProviderConfig.IKS.Enabled || os.Getenv("IKS_ENABLED") == "True") && icp.ProviderConfig.VPC.Encryption {
		apiKeyImp, err := utils.NewAPIKeyImpl(logger)
		if err != nil {
			logger.Error("Unable to create API key getter", zap.Reflect("Error", err))
			return err
		}
		logger.Info("Created NewAPIKeyImpl...")
		err = apiKeyImp.UpdateIAMKeys(icp.ProviderConfig)
		if err != nil {
			logger.Error("Unable to get API key", local.ZapError(err))
			return err
		}
		// If the retrieved API key is the same as previous one, returning error
		if vpcAPIKey == icp.ProviderConfig.VPC.G2APIKey {
			logger.Error("API key is not reset")
			return errors.New("API key is not found, reset the api key. If reset, retry after few minutes")
		}
		vpcBlockConfig.VPCConfig.APIKey = icp.ProviderConfig.VPC.G2APIKey
		vpcBlockConfig.VPCConfig.G2APIKey = icp.ProviderConfig.VPC.G2APIKey
	} else {
		conf := new(config.Config)
		configPath := filepath.Join(config.GetConfPathDir(), utils.ConfigFileName)
		_, err := toml.DecodeFile(configPath, conf)
		if err != nil {
			logger.Error("Failed to parse config file", zap.Error(err))
			return err
		}

		if vpcAPIKey == conf.VPC.G2APIKey {
			logger.Error("API is not reset")
			return errors.New("API key is not found, reset the api key. If reset, retry after few minutes")
		}
		icp.ProviderConfig.VPC.APIKey = conf.VPC.APIKey
		icp.ProviderConfig.VPC.G2APIKey = conf.VPC.G2APIKey
		vpcBlockConfig.VPCConfig.APIKey = conf.VPC.G2APIKey
		vpcBlockConfig.VPCConfig.G2APIKey = conf.VPC.G2APIKey
	}

	prov, err := icp.Registry.Get(icp.ProviderName)
	if err != nil {
		logger.Error("Not able to get the said provider, it might not registered", local.ZapError(err))
		return errors.New("unable to fetch the provider, hence unable to update the new API key")
	}

	err = prov.UpdateAPIKey(vpcBlockConfig, logger)
	if err != nil {
		logger.Error("Failed to update API key in the provider", local.ZapError(err))
		return errors.New("unable to update the new API key")
	}

	return nil
}
