package main

import (
	"fmt"
	"github.com/IBM/ibm-csi-common/cbr"
)

var (
	zoneID string
	ruleID string
)

func main() {

	var err error

	cbrContext := cbr.NewStorageCBR("xxx", "xxx", "xxx")

	cbrInput := cbr.CBR{
		VPC: []string{"xxx",
			"xxx"},
		Address:    []string{"169.23.56.234", "169.23.46.234", "169.23.22.0-169.23.22.255", "182.0.2.0/24"},
		ServiceRef: []string{},
	}

	//CBR Rule for IS service
	cbrInput.ServiceRef = []string{"containers-kubernetes"}
	zoneID, err = cbrContext.CreateCBRZone("E2E VPC Zone", cbrInput)

	if err != nil {
		fmt.Printf("Error creating zone: " + err.Error())
		return
	}

	ruleID, _ = cbrContext.CreateCBRRuleForISService(zoneID)
	if err != nil {
		fmt.Printf("Error creating rule: " + err.Error())
		return
	}

	cbrContext.DeleteCBRRuleZone(ruleID, zoneID)

	//CBR Rule for KMS service
	cbrInput.ServiceRef = []string{"server-protect"}
	zoneID, err = cbrContext.CreateCBRZone("E2E VPC Zone", cbrInput)

	if err != nil {
		fmt.Printf("Error creating zone: " + err.Error())
		return
	}

	ruleID, _ = cbrContext.CreateCBRRuleForKMSService(zoneID)
	if err != nil {
		fmt.Printf("Error creating rule: " + err.Error())
		return
	}

	cbrContext.DeleteCBRRuleZone(ruleID, zoneID)

	//CBR Rule for COS service
	zoneID, err = cbrContext.CreateCBRZone("E2E VPC Zone", cbrInput)

	if err != nil {
		fmt.Printf("Error creating zone: " + err.Error())
		return
	}

	ruleID, _ = cbrContext.CreateCBRRuleForCOSService(zoneID)
	if err != nil {
		fmt.Printf("Error creating rule: " + err.Error())
		return
	}

	cbrContext.DeleteCBRRuleZone(ruleID, zoneID)

	//CBR Rule for K8s service
	cbrInput.ServiceRef = []string{}
	zoneID, err = cbrContext.CreateCBRZone("E2E VPC Zone", cbrInput)

	if err != nil {
		fmt.Printf("Error creating zone: " + err.Error())
	} else {
		ruleID, err = cbrContext.CreateCBRRuleForContainerK8sService(zoneID)
		if err != nil {
			fmt.Printf("Error creating rule: " + err.Error())
		}
	}

	cbrContext.DeleteCBRRuleZone(ruleID, zoneID)
}
