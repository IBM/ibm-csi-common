# CBR library 

**1.) Inputs required** 
  
-  Function Account ID     
- Admin API key for function ID account.
- CBR Resource Group ID 
- List of VPCs from which access need to be restricted.
- List of address/subnet range from which access need to be restricted.

```
type cbr struct {
vpc []
address []
serviceRef [] 
}

```
**Methods**  
    AccountID , APIKey, resourceGroupId will be required to initialise the library.
   - NewCBRInterface( apiKey string, accountID string, resourceGroupID string )

 **Approach : User can create rules service wise on demand for single zone or multiple zones.** 

   - CreateCBRZone(name , input cbr ) zoneID.
   - CreateCBRForContainerK8sService( zoneID) ruleID
   - CreateCBRForISService( zoneID) ruleID
   - CreateCBRForKMSService( zoneID ) ruleID
   - CreateCBRForCOSService( zoneID ) ruleID
   - CreateCBRRule( zoneID, serviceName ) ruleID
   - DeleteCBRRuleZone (ruleID, zoneID)
