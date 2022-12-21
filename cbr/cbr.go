// Package cbr ...
package cbr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
)

const (
	kubernetes_service = "containers-kubernetes"
	is_service         = "is"
	cos_service        = "cloud-object-storage"
	keyProtect_service = "kms"
)

type CBR struct {
	VPC        []string
	Address    []string
	ServiceRef []string
}

// CBRInterface ...
type CBRInterface interface {

	// CreateCBRZone ...
	CreateCBRZone(name string, cbrInput CBR) (string, error)

	// CreateCBRRuleForContainerK8sService ...
	CreateCBRRuleForContainerK8sService(zoneID string) (string, error)

	// CreateCBRRuleForISService ...
	CreateCBRRuleForISService(zoneID string) (string, error)

	// CreateCBRRuleForKMSService ...
	CreateCBRRuleForKMSService(zoneID string) (string, error)

	// CreateCBRRuleForCOSService ...
	CreateCBRRuleForCOSService(zoneID string) (string, error)

	// CreateCBRRule ...
	CreateCBRRule(zoneID string, serviceName string) (string, error)

	// DeleteCBRRuleZone ...
	DeleteCBRRuleZone(ruleID string, zoneID string) error
}

// CBRInit ...
type CBRInit struct {
	accountID                       string
	resourceGroupID                 string
	contextBasedRestrictionsService *contextbasedrestrictionsv1.ContextBasedRestrictionsV1
}

// NewCBRInterface ...
func NewCBRInterface(apiKey string, accountID string, resourceGroupID string) CBRInterface {

	contextBasedRestrictionsServiceOptions := &contextbasedrestrictionsv1.ContextBasedRestrictionsV1Options{
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
		},
	}

	contextBasedRestrictionsService, err := contextbasedrestrictionsv1.NewContextBasedRestrictionsV1UsingExternalConfig(contextBasedRestrictionsServiceOptions)

	if err != nil {
		fmt.Printf("Error initializing contextbasedrestrictions sdk : " + err.Error())
		return nil
	}

	return &CBRInit{
		accountID:                       accountID,
		resourceGroupID:                 resourceGroupID,
		contextBasedRestrictionsService: contextBasedRestrictionsService,
	}
}

func (cbrInit *CBRInit) CreateCBRZone(name string, cbrInput CBR) (string, error) {
	fmt.Println("\nCreateZone() result:")

	// begin-create_zone
	var addressIntf []contextbasedrestrictionsv1.AddressIntf
	var vpcAddressModel *contextbasedrestrictionsv1.AddressVPC
	var ipAddressModel *contextbasedrestrictionsv1.AddressIPAddress
	var ipRangeAddressModel *contextbasedrestrictionsv1.AddressIPAddressRange
	var subnetAddressModel *contextbasedrestrictionsv1.AddressSubnet
	var serviceRefAddressModel *contextbasedrestrictionsv1.AddressServiceRef

	if len(cbrInput.Address) != 0 {
		for _, address := range cbrInput.Address {
			if strings.Contains(address, "-") { //If it is address range
				ipRangeAddressModel = &contextbasedrestrictionsv1.AddressIPAddressRange{
					Type:  core.StringPtr("ipRange"),
					Value: core.StringPtr(address),
				}

				addressIntf = append(addressIntf, ipRangeAddressModel)
			} else if strings.Contains(address, "/") { //If it is subnet
				subnetAddressModel = &contextbasedrestrictionsv1.AddressSubnet{
					Type:  core.StringPtr("subnet"),
					Value: core.StringPtr(address),
				}

				addressIntf = append(addressIntf, subnetAddressModel)
			} else { //If it is IPAddress

				ipAddressModel = &contextbasedrestrictionsv1.AddressIPAddress{
					Type:  core.StringPtr("ipAddress"),
					Value: core.StringPtr(address),
				}

				addressIntf = append(addressIntf, ipAddressModel)
			}
		}
	}

	if len(cbrInput.ServiceRef) != 0 {
		for _, serviceRef := range cbrInput.ServiceRef {
			serviceRefAddressModel = &contextbasedrestrictionsv1.AddressServiceRef{
				Type: core.StringPtr("serviceRef"),
				Ref: &contextbasedrestrictionsv1.ServiceRefValue{
					AccountID:   core.StringPtr(cbrInit.accountID),
					ServiceName: core.StringPtr(serviceRef),
				},
			}

			addressIntf = append(addressIntf, serviceRefAddressModel)
		}
	}

	if len(cbrInput.VPC) != 0 {
		for _, VPC := range cbrInput.VPC {
			vpcAddressModel = &contextbasedrestrictionsv1.AddressVPC{
				Type:  core.StringPtr("vpc"),
				Value: core.StringPtr(VPC),
			}

			addressIntf = append(addressIntf, vpcAddressModel)
		}
	}

	createZoneOptions := cbrInit.contextBasedRestrictionsService.NewCreateZoneOptions()
	createZoneOptions.SetName(name)
	createZoneOptions.SetAccountID(cbrInit.accountID)
	createZoneOptions.SetDescription("Zone-" + name)
	createZoneOptions.SetAddresses(addressIntf)

	zone, _, err := cbrInit.contextBasedRestrictionsService.CreateZone(createZoneOptions)
	if err != nil {
		return "", err
	}

	b, _ := json.MarshalIndent(zone, "", "  ")
	fmt.Println(string(b))

	// end-create_zone
	zoneID := *zone.ID

	return zoneID, nil
}

func (cbrInit *CBRInit) CreateCBRRuleForContainerK8sService(zoneID string) (string, error) {

	ruleID, err := cbrInit.CreateCBRRule(zoneID, kubernetes_service)

	return ruleID, err
}

func (cbrInit *CBRInit) CreateCBRRuleForISService(zoneID string) (string, error) {

	ruleID, err := cbrInit.CreateCBRRule(zoneID, is_service)

	return ruleID, err
}

func (cbrInit *CBRInit) CreateCBRRuleForKMSService(zoneID string) (string, error) {

	ruleID, err := cbrInit.CreateCBRRule(zoneID, keyProtect_service)

	return ruleID, err
}

func (cbrInit *CBRInit) CreateCBRRuleForCOSService(zoneID string) (string, error) {

	ruleID, err := cbrInit.CreateCBRRule(zoneID, cos_service)

	return ruleID, err
}

func (cbrInit *CBRInit) CreateCBRRule(zoneID string, serviceName string) (string, error) {
	fmt.Println("\nCreateRule() result:")
	// begin-create_rule

	ruleContextAttributeModel := &contextbasedrestrictionsv1.RuleContextAttribute{
		Name:  core.StringPtr("networkZoneId"),
		Value: core.StringPtr(zoneID),
	}

	ruleContextModel := &contextbasedrestrictionsv1.RuleContext{
		Attributes: []contextbasedrestrictionsv1.RuleContextAttribute{*ruleContextAttributeModel},
	}

	resourceModel := &contextbasedrestrictionsv1.Resource{
		Attributes: []contextbasedrestrictionsv1.ResourceAttribute{
			{
				Name:  core.StringPtr("accountId"),
				Value: core.StringPtr(cbrInit.accountID),
			},
			{
				Name:  core.StringPtr("serviceName"),
				Value: core.StringPtr(serviceName),
			},
			{
				Name:  core.StringPtr("resourceGroupId"),
				Value: core.StringPtr(cbrInit.resourceGroupID),
			},
		},
	}

	createRuleOptions := cbrInit.contextBasedRestrictionsService.NewCreateRuleOptions()
	createRuleOptions.SetDescription(serviceName + " rule")
	createRuleOptions.SetContexts([]contextbasedrestrictionsv1.RuleContext{*ruleContextModel})
	createRuleOptions.SetResources([]contextbasedrestrictionsv1.Resource{*resourceModel})
	createRuleOptions.SetEnforcementMode(contextbasedrestrictionsv1.CreateRuleOptionsEnforcementModeEnabledConst)
	rule, _, err := cbrInit.contextBasedRestrictionsService.CreateRule(createRuleOptions)
	if err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(rule, "", "  ")
	fmt.Println(string(b))

	// end-create_rule

	ruleID := *rule.ID

	return ruleID, nil
}

func (cbrInit *CBRInit) DeleteCBRRuleZone(ruleID string, zoneID string) error {
	var err error

	if len(ruleID) != 0 {
		// begin-delete_rule
		deleteRuleOptions := cbrInit.contextBasedRestrictionsService.NewDeleteRuleOptions(
			ruleID,
		)

		response, err := cbrInit.contextBasedRestrictionsService.DeleteRule(deleteRuleOptions)
		if err != nil {
			fmt.Printf("Error deleting rule : " + err.Error())
			return err
		}

		if response.StatusCode != 204 {
			fmt.Printf("\nUnexpected response status code received from DeleteRule(): %d\n", response.StatusCode)
		}

		// end-delete_rule
		fmt.Printf("\nDeleteRule() response status code: %d\n", response.StatusCode)
	}

	if len(zoneID) != 0 {
		// begin-delete_zone
		deleteZoneOptions := cbrInit.contextBasedRestrictionsService.NewDeleteZoneOptions(
			zoneID,
		)

		response, err := cbrInit.contextBasedRestrictionsService.DeleteZone(deleteZoneOptions)
		if err != nil {
			fmt.Printf("Error deleting rule : " + err.Error())
			return err
		}

		if response.StatusCode != 204 {
			fmt.Printf("\nUnexpected response status code received from DeleteZone(): %d\n", response.StatusCode)
		}

		// end-delete_zone
		fmt.Printf("\nDeleteZone() response status code: %d\n", response.StatusCode)
	}

	return err
}
