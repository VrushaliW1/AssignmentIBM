

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"reflect"
	"strings"
	"time"
	 "sort"

)

// For now, We will not use go generate here : go:generate go run scripts/generate_go_schema.go
// This is because there are different assets of various structures coming in. The only
// certainity is that the assets should have an asset id and asset type. These will be extracted
// and the asset data stored and manipulated as such
// Alerts have been temporarily written in a manner that they will run only for applicable types -
// essentially based on whether the observed values are in the incoming data stream
// This needs to be modified as well.
// Major revamp required to the approach based on the use case.

//***************************************************
const MYVERSION string = "1.0"
//***************************************************
//* CONTRACT initialization and runtime engine
//***************************************************
// DEFAULTNICKNAME is used when a contract is initialized without giving it a nickname
const DEFAULTNICKNAME string = "BUILDING" 

// CONTRACTSTATEKEY is used to store contract state, including version, nickname and activeAssets
const CONTRACTSTATEKEY string = "ContractStateKey"

// ContractState struct defines contract state. Unlike the main contract maps, structs work fine
// for this fixed structure.
type ContractState struct {
	Version      string           `json:"version"`
    Nickname     string           `json:"nickname"`
	ActiveAssets map[string]bool  `json:"activeAssets"`
	ActiveAccounts map[string]bool  `json:"activeAccounts"`
}
//*************************************************** Recent 
// RECENTSTATESKEY is used as key for recent states bucket
const RECENTSTATESKEY string = "RecentStatesKey"

// RecentStates is JSON encoded string slice 
type RecentStates struct {
    RecentStates []string `json:"recentStates"`
}

// AssetIDT is assetID as type, used for simple unmarshaling
type AssetIDT struct {
    ID string `json:"assetID"`
} 
// AccountIDT is accountID as type, used for simple unmarshaling
type AccountIDT struct {
    ID string `json:"accountID"`
} 
// MaxRecentStates is an arbitrary limit on how many asset states we track across the 
// entire contract
const MaxRecentStates int = 20
///********************** Map ******************
var CASESENSITIVEMODE bool = false
///************************Logger*******************************
type LogLevel int

const (
	// CRITICAL means cannot function
    CRITICAL LogLevel = iota
    // ERROR means something is wrong
	ERROR
    // WARNING means something might be wrong
	WARNING
    // NOTICE means take note, this should be investigated
	NOTICE
    // INFO means this happened and might be of interest
	INFO
    // DEBUG allows for a peek into the guts of the app for debugging
	DEBUG
)

var logLevelNames = []string {
	"CRITICAL",
	"ERROR",
	"WARNING",
	"NOTICE",
	"INFO",
	"DEBUG",
}

// DEFAULTLOGGINGLEVEL is normally INFO in test and WARNING in production
const DEFAULTLOGGINGLEVEL = DEBUG

// ContractLogger is our version of goLogger
type ContractLogger struct {
    module      string
   level       LogLevel
}

// ILogger the goLogger interface to which we are 100% compatible
type ILogger interface {
    Critical(args ...interface{})
    Criticalf(format string, args ...interface{})
    Error(args ...interface{})
    Errorf(format string, args ...interface{})
    Warning(args ...interface{}) 
    Warningf(format string, args ...interface{})
    Notice(args ...interface{})
    Noticef(format string, args ...interface{})
    Info(args ...interface{})
    Infof(format string, args ...interface{})
    Debug(args ...interface{})
    Debugf(format string, args ...interface{})
}


// ************************************
// definitions
// ************************************

// SimpleChaincode is the receiver for all shim API
type SimpleChaincode struct {
}

// ASSETID is the JSON tag for the assetID
const ASSETID string = "assetID"

// ASSETTYPE is the JSON tag for the asset type
const ASSETTYPE string = "assettype"

// ASSETNAME Asset description from which type is inferred
const ASSETNAME string = "name"

// ACCOUNTID is the JSON tag for the assetID
const ACCOUNTID string = "accountID"

//// ACCOUNTNAME Asset description from which type is inferred
const ACCOUNTNAME string = "acname"

// TIMESTAMP is the JSON tag for timestamps, devices must use this tag to be compatible!
const TIMESTAMP string = "timestamp"

// ArgsMap is a generic map[string]interface{} to be used as a receiver
type ArgsMap map[string]interface{}

var log = NewContractLogger(DEFAULTNICKNAME, DEFAULTLOGGINGLEVEL)

// ************************************
// start the message pumps
// ************************************
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		log.Infof("ERROR starting Simple Chaincode: %s", err)
	}
}

// Init is called in deploy mode when contract is initialized
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var stateArg ContractState
	var err error

	log.Info("Entering INIT")

	if len(args) != 1 {
		err = errors.New("init expects one argument, a JSON string with  mandatory version and optional nickname")
		log.Critical(err)
		return nil, err
	}

	err = json.Unmarshal([]byte(args[0]), &stateArg)
	if err != nil {
		err = fmt.Errorf("Version argument unmarshal failed: %s", err)
		log.Critical(err)
		return nil, err
	}

	if stateArg.Nickname == "" {
		stateArg.Nickname = DEFAULTNICKNAME
	}

	(*log).setModule(stateArg.Nickname)

	err = initializeContractState(stub, stateArg.Version, stateArg.Nickname)
	if err != nil {
		return nil, err
	}

	log.Info("Contract initialized")
	return nil, nil
}

// Invoke is called in invoke mode to delegate state changing function messages
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if function == "createAsset" {
		return t.createAsset(stub, args)
	} else if function == "updateAsset" {
		return t.updateAsset(stub, args)
	} else if function == "deleteAsset" {
		return t.deleteAsset(stub, args)
	} else if function == "deleteAllAssets" {
		return t.deleteAllAssets(stub, args)
	} else if function == "deletePropertiesFromAsset" {
		return t.deletePropertiesFromAsset(stub, args)
	} else if function == "setLoggingLevel" {
		return nil, t.setLoggingLevel(stub, args)
	} else if function == "setCreateOnUpdate" {
		return nil, t.setCreateOnUpdate(stub, args)
	}else if function == "createAccount" {
		return  t.createAccount(stub, args)
	}
	err := fmt.Errorf("Invoke received unknown invocation: %s", function)
	log.Warning(err)
	return nil, err
}

// Query is called in query mode to delegate non-state-changing queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if function == "readAsset" {
		return t.readAsset(stub, args)
	} else if function == "readAllAssets" {
		return t.readAllAssets(stub, args)
	} else if function == "readRecentStates" {
		return readRecentStates(stub)
	} else if function == "readAssetHistory" {
		return t.readAssetHistory(stub, args)
	} else if function == "readContractObjectModel" {
		return t.readContractObjectModel(stub, args)
	} else if function == "readContractState" {
		return t.readContractState(stub, args)
	} else if function == "readAllAccounts" {
		return t.readAllAccounts(stub, args)
	}
	// To be added
	/*   else if function == "readAllAssetsOfType" {
	return t.readAllAssetsOfType(stub, args)*/
	err := fmt.Errorf("Query received unknown invocation: %s", function)
	log.Warning(err)
	return nil, err
}

//***************************************************
//***************************************************
//* ASSET CRUD INTERFACE
//***************************************************
//***************************************************

// ************************************
// createAsset
// ************************************
func (t *SimpleChaincode) createAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string
	var assetType string
	var assetName string
	var argsMap ArgsMap
	var event interface{}
	var found bool
	var err error
	//var timeIn time.Time

	log.Info("Entering createAsset")

	// allowing 2 args because updateAsset is allowed to redirect when
	// asset does not exist
	if len(args) < 1 || len(args) > 2 {
		err = errors.New("Expecting one JSON event object")
		log.Error(err)
		return nil, err
	}

	assetID = ""
	assetType = ""
	assetName = ""
	eventBytes := []byte(args[0])
	log.Debugf("createAsset arg: %s", args[0])

	err = json.Unmarshal(eventBytes, &event)
	if err != nil {
		log.Errorf("createAsset failed to unmarshal arg: %s", err)
		return nil, err
	}

	if event == nil {
		err = errors.New("createAsset unmarshal arg created nil event")
		log.Error(err)
		return nil, err
	}

	argsMap, found = event.(map[string]interface{})
	if !found {
		err := errors.New("createAsset arg is not a map shape")
		log.Error(err)
		return nil, err
	}

	// is assetID present or blank?
	assetIDBytes, found := getObject(argsMap, ASSETID)
	if found {
		assetID, found = assetIDBytes.(string)
		if !found || assetID == "" {
			err := errors.New("createAsset arg does not include assetID ")
			log.Error(err)
			return nil, err
		}
	}
	// Is asset name present?
	assetTypeBytes, found := getObject(argsMap, ASSETNAME)
	if found {
		assetName, found = assetTypeBytes.(string)
		if !found || assetName == "" {
			err := errors.New("createAsset arg does not include assetName ")
			log.Error(err)
			return nil, err
		}
	}
	if strings.Contains(assetName, "Plug") {
		assetType = "smartplug"
	} else {
		assetType = "motor"
	}

	log.Info(assetType)
	sAssetKey := assetID + "_" + assetType
	found = assetIsActive(stub, sAssetKey)
	if found {
		err := fmt.Errorf("createAsset arg asset %s of type %s already exists", assetID, assetType)
		log.Error(err)
		return nil, err
	}

	// For now, timestamp is being sent in from the invocation to the contract
	// Once the BlueMix instance supports GetTxnTimestamp, we will incorporate the
	// changes to the contract

	// run the rules and raise or clear alerts
	alerts := newAlertStatus()
	if argsMap.executeRules(&alerts) {
		// NOT compliant!
		log.Noticef("createAsset assetID %s of type %s is noncompliant", assetID, assetType)
		argsMap["alerts"] = alerts
		delete(argsMap, "incompliance")
	} else {
		if alerts.AllClear() {
			// all false, no need to appear
			delete(argsMap, "alerts")
		} else {
			argsMap["alerts"] = alerts
		}
		argsMap["incompliance"] = true
	}

	// copy incoming event to outgoing state
	// this contract respects the fact that createAsset can accept a partial state
	// as the moral equivalent of one or more discrete events
	// further: this contract understands that its schema has two discrete objects
	// that are meant to be used to send events: common, and custom
	stateOut := argsMap

	// save the original event
	stateOut["lastEvent"] = make(map[string]interface{})
	stateOut["lastEvent"].(map[string]interface{})["function"] = "createAsset"
	stateOut["lastEvent"].(map[string]interface{})["args"] = args[0]
	if len(args) == 2 {
		// in-band protocol for redirect
		stateOut["lastEvent"].(map[string]interface{})["redirectedFromFunction"] = args[1]
	}

	// marshal to JSON and write
	stateJSON, err := json.Marshal(&stateOut)
	if err != nil {
		err := fmt.Errorf("createAsset state for assetID %s failed to marshal", assetID)
		log.Error(err)
		return nil, err
	}

	// finally, put the new state
	log.Infof("Putting new asset state %s to ledger", string(stateJSON))
	// The key i 'assetid'_'type'

	err = stub.PutState(sAssetKey, []byte(stateJSON))
	if err != nil {
		err = fmt.Errorf("createAsset AssetID %s of Type %s PUTSTATE failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}
	log.Infof("createAsset AssetID %s of type %s state successfully written to ledger: %s", assetID, assetType, string(stateJSON))

	// add asset to contract state
	err = addAssetToContractState(stub, sAssetKey)
	if err != nil {
		err := fmt.Errorf("createAsset asset %s of type %s failed to write asset state: %s", assetID, assetType, err)
		log.Critical(err)
		return nil, err
	}

	err = pushRecentState(stub, string(stateJSON),"0")
	if err != nil {
		err = fmt.Errorf("createAsset AssetID %s of type %s push to recentstates failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// save state history
	err = createStateHistory(stub, sAssetKey, string(stateJSON))
	if err != nil {
		err := fmt.Errorf("createAsset asset %s of type %s state history save failed: %s", assetID, sAssetKey, err)
		log.Critical(err)
		return nil, err
	}
	return nil, nil
}

// ************************************
// updateAsset
// ************************************
func (t *SimpleChaincode) updateAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string
	var assetType string
	var assetName string
	var argsMap ArgsMap
	var event interface{}
	var ledgerMap ArgsMap
	var ledgerBytes interface{}
	var found bool
	var err error
	//var timeIn time.Time

	log.Info("Entering updateAsset")

	if len(args) != 1 {
		err = errors.New("Expecting one JSON event object")
		log.Error(err)
		return nil, err
	}

	assetID = ""
	assetType = ""
	assetName = ""
	eventBytes := []byte(args[0])
	log.Debugf("updateAsset arg: %s", args[0])

	err = json.Unmarshal(eventBytes, &event)
	if err != nil {
		log.Errorf("updateAsset failed to unmarshal arg: %s", err)
		return nil, err
	}

	if event == nil {
		err = errors.New("createAsset unmarshal arg created nil event")
		log.Error(err)
		return nil, err
	}

	argsMap, found = event.(map[string]interface{})
	if !found {
		err := errors.New("updateAsset arg is not a map shape")
		log.Error(err)
		return nil, err
	}

	// is assetID present or blank?
	assetIDBytes, found := getObject(argsMap, ASSETID)
	if found {
		assetID, found = assetIDBytes.(string)
		if !found || assetID == "" {
			err := errors.New("updateAsset arg does not include assetID")
			log.Error(err)
			return nil, err
		}
	}
	log.Noticef("updateAsset found assetID %s", assetID)

	// Is asset name present?
	assetTypeBytes, found := getObject(argsMap, ASSETNAME)
	if found {
		assetName, found = assetTypeBytes.(string)
		if !found || assetName == "" {
			err := errors.New("createAsset arg does not include assetName ")
			log.Error(err)
			return nil, err
		}
	}
	if strings.Contains(assetName, "Plug") {
		assetType = "smartplug"
	} else {
		assetType = "motor"
	}
	log.Noticef("updateAsset found assetID %s of type %s ", assetID, assetType)

	sAssetKey := assetID + "_" + assetType
	found = assetIsActive(stub, sAssetKey)
	if !found {
		// redirect to createAsset with same parameter list
		if canCreateOnUpdate(stub) {
			log.Noticef("updateAsset redirecting asset %s of type %s to createAsset", assetID, assetType)
			var newArgs = []string{args[0], "updateAsset"}
			return t.createAsset(stub, newArgs)
		}
		err = fmt.Errorf("updateAsset asset %s of type %s does not exist", assetID, assetType)
		log.Error(err)
		return nil, err
	}
	// For now, timestamp is being sent in from the invocation to the contract
	// Once the BlueMix instance supports GetTxnTimestamp, we will incorporate the
	// changes to the contract

	// **********************************
	// find the asset state in the ledger
	// **********************************
	log.Infof("updateAsset: retrieving asset %s state from ledger", sAssetKey)
	assetBytes, err := stub.GetState(sAssetKey)
	if err != nil {
		log.Errorf("updateAsset assetID %s of type %s GETSTATE failed: %s", assetID, assetType, err)
		return nil, err
	}

	// unmarshal the existing state from the ledger to theinterface
	err = json.Unmarshal(assetBytes, &ledgerBytes)
	if err != nil {
		log.Errorf("updateAsset assetID %s of type %s unmarshal failed: %s", assetID, assetType, err)
		return nil, err
	}

	// assert the existing state as a map
	ledgerMap, found = ledgerBytes.(map[string]interface{})
	if !found {
		log.Errorf("updateAsset assetID %s of type %s LEDGER state is not a map shape", assetID, assetType)
		return nil, err
	}

	// now add incoming map values to existing state to merge them
	// this contract respects the fact that updateAsset can accept a partial state
	// as the moral equivalent of one or more discrete events
	// further: this contract understands that its schema has two discrete objects
	// that are meant to be used to send events: common, and custom
	// ledger has to have common section
	stateOut := deepMerge(map[string]interface{}(argsMap),
		map[string]interface{}(ledgerMap))
	log.Debugf("updateAsset assetID %s merged state: %s of type %s", assetID, assetType, stateOut)

	// handle compliance section
	alerts := newAlertStatus()
	a, found := stateOut["alerts"] // is there an existing alert state?
	if found {
		// convert to an AlertStatus, which does not work by type assertion
		log.Debugf("updateAsset Found existing alerts state: %s", a)
		// complex types are all untyped interfaces, so require conversion to
		// the structure that is used, but not in the other direction as the
		// type is properly specified
		alerts.alertStatusFromMap(a.(map[string]interface{}))
	}
	// important: rules need access to the entire calculated state
	if ledgerMap.executeRules(&alerts) {
		// true means noncompliant
		log.Noticef("updateAsset assetID %s of type %s is noncompliant", assetID, assetType)
		// update ledger with new state, if all clear then delete
		stateOut["alerts"] = alerts
		delete(stateOut, "incompliance")
	} else {
		if alerts.AllClear() {
			// all false, no need to appear
			delete(stateOut, "alerts")
		} else {
			stateOut["alerts"] = alerts
		}
		stateOut["incompliance"] = true
	}

	// save the original event
	stateOut["lastEvent"] = make(map[string]interface{})
	stateOut["lastEvent"].(map[string]interface{})["function"] = "updateAsset"
	stateOut["lastEvent"].(map[string]interface{})["args"] = args[0]

	// Write the new state to the ledger
	stateJSON, err := json.Marshal(ledgerMap)
	if err != nil {
		err = fmt.Errorf("updateAsset AssetID %s of type %s marshal failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// finally, put the new state
	err = stub.PutState(sAssetKey, []byte(stateJSON))
	if err != nil {
		err = fmt.Errorf("updateAsset AssetID %s of type %s PUTSTATE failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}
	err = pushRecentState(stub, string(stateJSON),"0")
	if err != nil {
		err = fmt.Errorf("updateAsset AssetID %s push to recentstates failed: %s", assetID, err)
		log.Error(err)
		return nil, err
	}

	// add history state
	err = updateStateHistory(stub, sAssetKey, string(stateJSON))
	if err != nil {
		err = fmt.Errorf("updateAsset AssetID %s of type %s push to history failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// NOTE: Contract state is not updated by updateAsset

	return nil, nil
}

// ************************************
// deleteAsset
// ************************************
func (t *SimpleChaincode) deleteAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string
	var assetType string
	var assetName string
	var argsMap ArgsMap
	var event interface{}
	var found bool
	var err error

	if len(args) != 1 {
		err = errors.New("Expecting one JSON state object with an assetID")
		log.Error(err)
		return nil, err
	}

	assetID = ""
	assetType = ""
	assetName = ""
	eventBytes := []byte(args[0])
	log.Debugf("deleteAsset arg: %s", args[0])

	err = json.Unmarshal(eventBytes, &event)
	if err != nil {
		log.Errorf("deleteAsset failed to unmarshal arg: %s", err)
		return nil, err
	}

	argsMap, found = event.(map[string]interface{})
	if !found {
		err := errors.New("deleteAsset arg is not a map shape")
		log.Error(err)
		return nil, err
	}

	// is assetID present or blank?
	assetIDBytes, found := getObject(argsMap, ASSETID)
	if found {
		assetID, found = assetIDBytes.(string)
		if !found || assetID == "" {
			err := errors.New("deleteAsset arg does not include assetID")
			log.Error(err)
			return nil, err
		}
	}

	// Is asset name present?
	assetTypeBytes, found := getObject(argsMap, ASSETNAME)
	if found {
		assetName, found = assetTypeBytes.(string)
		if !found || assetName == "" {
			err := errors.New("createAsset arg does not include assetName ")
			log.Error(err)
			return nil, err
		}
	}
	if strings.Contains(assetName, "Plug") {
		assetType = "smartplug"
	} else {
		assetType = "motor"
	}
	sAssetKey := assetID + "_" + assetType
	found = assetIsActive(stub, sAssetKey)
	if !found {
		err = fmt.Errorf("deleteAsset assetID %s of type  %s does not exist", assetID, assetType)
		log.Error(err)
		return nil, err
	}

	// Delete the key / asset from the ledger
	err = stub.DelState(sAssetKey)
	if err != nil {
		log.Errorf("deleteAsset assetID %s of type %s failed DELSTATE", assetID, assetType)
		return nil, err
	}
	// remove asset from contract state
	err = removeAssetFromContractState(stub, sAssetKey)
	if err != nil {
		err := fmt.Errorf("deleteAsset asset %s of type %s failed to remove asset from contract state: %s", assetID, assetType, err)
		log.Critical(err)
		return nil, err
	}
	// save state history
	err = deleteStateHistory(stub, sAssetKey)
	if err != nil {
		err := fmt.Errorf("deleteAsset asset %s of type %s state history delete failed: %s", assetID, assetType, err)
		log.Critical(err)
		return nil, err
	}
	// push the recent state
	err = removeAssetFromRecentState(stub, sAssetKey)
	if err != nil {
		err := fmt.Errorf("deleteAsset asset %s recent state removal failed: %s", assetID, assetType, err)
		log.Critical(err)
		return nil, err
	}

	return nil, nil
}

// ************************************
// deletePropertiesFromAsset
// ************************************
func (t *SimpleChaincode) deletePropertiesFromAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string
	var assetType string
	var assetName string
	var argsMap ArgsMap
	var event interface{}
	var ledgerMap ArgsMap
	var ledgerBytes interface{}
	var found bool
	var err error
	var alerts AlertStatus

	if len(args) < 1 {
		err = errors.New("Not enough arguments. Expecting one JSON object with mandatory AssetID and property name array")
		log.Error(err)
		return nil, err
	}
	eventBytes := []byte(args[0])

	err = json.Unmarshal(eventBytes, &event)
	if err != nil {
		log.Error("deletePropertiesFromAsset failed to unmarshal arg")
		return nil, err
	}

	argsMap, found = event.(map[string]interface{})
	if !found {
		err := errors.New("updateAsset arg is not a map shape")
		log.Error(err)
		return nil, err
	}
	log.Debugf("deletePropertiesFromAsset arg: %+v", argsMap)

	// is assetID present or blank?
	assetIDBytes, found := getObject(argsMap, ASSETID)
	if found {
		assetID, found = assetIDBytes.(string)
		if !found || assetID == "" {
			err := errors.New("deletePropertiesFromAsset arg does not include assetID")
			log.Error(err)
			return nil, err
		}
	}
	// Is asset name present?
	assetTypeBytes, found := getObject(argsMap, ASSETNAME)
	if found {
		assetName, found = assetTypeBytes.(string)
		if !found || assetName == "" {
			err := errors.New("createAsset arg does not include assetName ")
			log.Error(err)
			return nil, err
		}
	}
	if strings.Contains(assetName, "Plug") {
		assetType = "smartplug"
	} else {
		assetType = "motor"
	}
	sAssetKey := assetID + "_" + assetType

	found = assetIsActive(stub, sAssetKey)
	if !found {
		err = fmt.Errorf("deletePropertiesFromAsset assetID %s of type %s does not exist", assetID, assetType)
		log.Error(err)
		return nil, err
	}

	// is there a list of property names?
	var qprops []interface{}
	qpropsBytes, found := getObject(argsMap, "qualPropsToDelete")
	if found {
		qprops, found = qpropsBytes.([]interface{})
		log.Debugf("deletePropertiesFromAsset qProps: %+v, Found: %+v, Type: %+v", qprops, found, reflect.TypeOf(qprops))
		if !found || len(qprops) < 1 {
			log.Errorf("deletePropertiesFromAsset asset %s of type %s qualPropsToDelete is not an array or is empty", assetID, assetType)
			return nil, err
		}
	} else {
		log.Errorf("deletePropertiesFromAsset asset %s of type %s has no qualPropsToDelete argument", assetID, assetType)
		return nil, err
	}

	// **********************************
	// find the asset state in the ledger
	// **********************************
	log.Infof("deletePropertiesFromAsset: retrieving asset %s of type %s state from ledger", assetID, assetType)
	assetBytes, err := stub.GetState(sAssetKey)
	if err != nil {
		err = fmt.Errorf("deletePropertiesFromAsset AssetID %s of type %s GETSTATE failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// unmarshal the existing state from the ledger to the interface
	err = json.Unmarshal(assetBytes, &ledgerBytes)
	if err != nil {
		err = fmt.Errorf("deletePropertiesFromAsset AssetID %s of type %s unmarshal failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// assert the existing state as a map
	ledgerMap, found = ledgerBytes.(map[string]interface{})
	if !found {
		err = fmt.Errorf("deletePropertiesFromAsset AssetID %s of type %s LEDGER state is not a map shape", assetID, assetType)
		log.Error(err)
		return nil, err
	}

	// now remove properties from state, they are qualified by level
OUTERDELETELOOP:
	for p := range qprops {
		prop := qprops[p].(string)
		log.Debugf("deletePropertiesFromAsset AssetID %s of type %s deleting qualified property: %s", assetID, assetType, prop)
		// TODO Ugly, isolate in a function at some point
		if (CASESENSITIVEMODE && strings.HasSuffix(prop, ASSETID)) ||
			(!CASESENSITIVEMODE && strings.HasSuffix(strings.ToLower(prop), strings.ToLower(ASSETID)) ||
				CASESENSITIVEMODE && strings.HasSuffix(prop, ASSETTYPE)) ||
			(!CASESENSITIVEMODE && strings.HasSuffix(strings.ToLower(prop), strings.ToLower(ASSETTYPE))) {
			log.Warningf("deletePropertiesFromAsset AssetID %s of type %s cannot delete protected qualified property: %s or type %s", assetID, assetType, prop)
		} else {
			levels := strings.Split(prop, ".")
			lm := (map[string]interface{})(ledgerMap)
			for l := range levels {
				// lev is the name of a level
				lev := levels[l]
				if l == len(levels)-1 {
					// we're here, delete the actual property name from this level of the map
					levActual, found := findMatchingKey(lm, lev)
					if !found {
						log.Warningf("deletePropertiesFromAsset AssetID %s of type %s property match %s not found", assetID, assetType, lev)
						continue OUTERDELETELOOP
					}
					log.Debugf("deletePropertiesFromAsset AssetID %s of type %s deleting %s", assetID, assetType, prop)
					delete(lm, levActual)
				} else {
					// navigate to the next level object
					log.Debugf("deletePropertiesFromAsset AssetID %s of type %s navigating to level %s", assetID, assetType, lev)
					lmBytes, found := findObjectByKey(lm, lev)
					if found {
						lm, found = lmBytes.(map[string]interface{})
						if !found {
							log.Noticef("deletePropertiesFromAsset AssetID %s of type %s level %s not found in ledger", assetID, assetType, lev)
							continue OUTERDELETELOOP
						}
					}
				}
			}
		}
	}
	log.Debugf("updateAsset AssetID %s final state: %s of type %s ", assetID, assetType, ledgerMap)

	// set timestamp
	// TODO timestamp from the stub - GetTxnTimestamp
	ledgerMap[TIMESTAMP] = time.Now()

	// handle compliance section
	alerts = newAlertStatus()
	a, found := argsMap["alerts"] // is there an existing alert state?
	if found {
		// convert to an AlertStatus, which does not work by type assertion
		log.Debugf("deletePropertiesFromAsset Found existing alerts state: %s", a)
		// complex types are all untyped interfaces, so require conversion to
		// the structure that is used, but not in the other direction as the
		// type is properly specified
		alerts.alertStatusFromMap(a.(map[string]interface{}))
	}
	// important: rules need access to the entire calculated state
	if ledgerMap.executeRules(&alerts) {
		// true means noncompliant
		log.Noticef("deletePropertiesFromAsset assetID %s of type %s is noncompliant", assetID, assetType)
		// update ledger with new state, if all clear then delete
		ledgerMap["alerts"] = alerts
		delete(ledgerMap, "incompliance")
	} else {
		if alerts.AllClear() {
			// all false, no need to appear
			delete(ledgerMap, "alerts")
		} else {
			ledgerMap["alerts"] = alerts
		}
		ledgerMap["incompliance"] = true
	}

	// save the original event
	ledgerMap["lastEvent"] = make(map[string]interface{})
	ledgerMap["lastEvent"].(map[string]interface{})["function"] = "deletePropertiesFromAsset"
	ledgerMap["lastEvent"].(map[string]interface{})["args"] = args[0]

	// Write the new state to the ledger
	stateJSON, err := json.Marshal(ledgerMap)
	if err != nil {
		err = fmt.Errorf("deletePropertiesFromAsset AssetID %s of type %s marshal failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// finally, put the new state
	err = stub.PutState(sAssetKey, []byte(stateJSON))
	if err != nil {
		err = fmt.Errorf("deletePropertiesFromAsset AssetID %s of type %s PUTSTATE failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}
	err = pushRecentState(stub, string(stateJSON),"0")
	if err != nil {
		err = fmt.Errorf("deletePropertiesFromAsset AssetID %s of type %s push to recentstates failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// add history state
	err = updateStateHistory(stub, sAssetKey, string(stateJSON))
	if err != nil {
		err = fmt.Errorf("deletePropertiesFromAsset AssetID %s of type %s push to history failed: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	return nil, nil
}

// ************************************
// deletaAllAssets
// ************************************
func (t *SimpleChaincode) deleteAllAssets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var sAssetKey string
	var err error

	if len(args) > 0 {
		err = errors.New("Too many arguments. Expecting none.")
		log.Error(err)
		return nil, err
	}

	aa, err := getActiveAssets(stub)
	if err != nil {
		err = fmt.Errorf("deleteAllAssets failed to get the active assets: %s", err)
		log.Error(err)
		return nil, err
	}
	for i := range aa {
		sAssetKey = aa[i]

		// Delete the key / asset from the ledger
		err = stub.DelState(sAssetKey)
		if err != nil {
			err = fmt.Errorf("deleteAllAssets arg %d AssetKey %s failed DELSTATE", i, sAssetKey)
			log.Error(err)
			return nil, err
		}
		// remove asset from contract state
		err = removeAssetFromContractState(stub, sAssetKey)
		if err != nil {
			err = fmt.Errorf("deleteAllAssets asset %s failed to remove asset from contract state: %s", sAssetKey, err)
			log.Critical(err)
			return nil, err
		}
		// save state history
		err = deleteStateHistory(stub, sAssetKey)
		if err != nil {
			err := fmt.Errorf("deleteAllAssets asset %s state history delete failed: %s", sAssetKey, err)
			log.Critical(err)
			return nil, err
		}
	}
	err = clearRecentStates(stub)
	if err != nil {
		err = fmt.Errorf("deleteAllAssets clearRecentStates failed: %s", err)
		log.Error(err)
		return nil, err
	}
	return nil, nil
}

// ************************************
// readAsset
// ************************************
func (t *SimpleChaincode) readAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string
	var assetType string
	var assetName string
	var argsMap ArgsMap
	var request interface{}
	var assetBytes []byte
	var found bool
	var err error

	if len(args) != 1 {
		err = errors.New("Expecting one JSON event object")
		log.Error(err)
		return nil, err
	}

	requestBytes := []byte(args[0])
	log.Debugf("readAsset arg: %s", args[0])

	err = json.Unmarshal(requestBytes, &request)
	if err != nil {
		log.Errorf("readAsset failed to unmarshal arg: %s", err)
		return nil, err
	}

	argsMap, found = request.(map[string]interface{})
	if !found {
		err := errors.New("readAsset arg is not a map shape")
		log.Error(err)
		return nil, err
	}

	// is assetID present or blank?
	assetIDBytes, found := getObject(argsMap, ASSETID)
	if found {
		assetID, found = assetIDBytes.(string)
		if !found || assetID == "" {
			err := errors.New("readAsset arg does not include assetID")
			log.Error(err)
			return nil, err
		}
	}
	// Is asset name present?
	assetTypeBytes, found := getObject(argsMap, ASSETNAME)
	if found {
		assetName, found = assetTypeBytes.(string)
		if !found || assetName == "" {
			err := errors.New("createAsset arg does not include assetName ")
			log.Error(err)
			return nil, err
		}
	}
	sMsg := "Inside readAsset assetName: " + assetName
	log.Info(sMsg)
	if strings.Contains(assetName, "Plug") {
		assetType = "smartplug"
	} else {
		assetType = "motor"
	}
	sMsgTyoe := "Inside readAsset assetType: " + assetType
	log.Info(sMsgTyoe)
	sAssetKey := assetID + "_" + assetType
	found = assetIsActive(stub, sAssetKey)
	if !found {
		err := fmt.Errorf("readAsset arg asset %s of type %s does not exist", assetID, assetType)
		log.Error(err)
		return nil, err
	}

	// Get the state from the ledger
	assetBytes, err = stub.GetState(sAssetKey)
	if err != nil {
		log.Errorf("readAsset assetID %s of type %s failed GETSTATE", assetID, assetType)
		return nil, err
	}

	return assetBytes, nil
}

// ************************************
// readAllAssets
// ************************************
func (t *SimpleChaincode) readAllAssets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var sAssetKey string
	var err error
	var results []interface{}
	var state interface{}

	if len(args) > 0 {
		err = errors.New("readAllAssets expects no arguments")
		log.Error(err)
		return nil, err
	}

	aa, err := getActiveAssets(stub)
	if err != nil {
		err = fmt.Errorf("readAllAssets failed to get the active assets: %s", err)
		log.Error(err)
		return nil, err
	}
	results = make([]interface{}, 0, len(aa))
	for i := range aa {
		sAssetKey = aa[i]
		// Get the state from the ledger
		assetBytes, err := stub.GetState(sAssetKey)
		if err != nil {
			// best efforts, return what we can
			log.Errorf("readAllAssets assetID %s failed GETSTATE", sAssetKey)
			continue
		} else {
			err = json.Unmarshal(assetBytes, &state)
			if err != nil {
				// best efforts, return what we can
				log.Errorf("readAllAssets assetID %s failed to unmarshal", sAssetKey)
				continue
			}
			results = append(results, state)
		}
	}

	resultsStr, err := json.Marshal(results)
	if err != nil {
		err = fmt.Errorf("readallAssets failed to marshal results: %s", err)
		log.Error(err)
		return nil, err
	}

	return []byte(resultsStr), nil
}

// ************************************
// readAssetHistory
// ************************************
func (t *SimpleChaincode) readAssetHistory(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetBytes []byte
	var assetID string
	var assetType string
	var assetName string
	var argsMap ArgsMap
	var request interface{}
	var found bool
	var err error

	if len(args) != 1 {
		err = errors.New("readAssetHistory expects a JSON encoded object with assetID and count")
		log.Error(err)
		return nil, err
	}

	requestBytes := []byte(args[0])
	log.Debugf("readAssetHistory arg: %s", args[0])

	err = json.Unmarshal(requestBytes, &request)
	if err != nil {
		err = fmt.Errorf("readAssetHistory failed to unmarshal arg: %s", err)
		log.Error(err)
		return nil, err
	}

	argsMap, found = request.(map[string]interface{})
	if !found {
		err := errors.New("readAssetHistory arg is not a map shape")
		log.Error(err)
		return nil, err
	}

	// is assetID present or blank?
	assetIDBytes, found := getObject(argsMap, ASSETID)
	if found {
		assetID, found = assetIDBytes.(string)
		if !found || assetID == "" {
			err := errors.New("readAssetHistory arg does not include assetID")
			log.Error(err)
			return nil, err
		}
	}
	// Is asset name present?
	assetTypeBytes, found := getObject(argsMap, ASSETNAME)
	if found {
		assetName, found = assetTypeBytes.(string)
		if !found || assetName == "" {
			err := errors.New("createAsset arg does not include assetName ")
			log.Error(err)
			return nil, err
		}
	}
	if strings.Contains(assetName, "Plug") {
		assetType = "smartplug"
	} else {
		assetType = "motor"
	}
	sAssetKey := assetID + "_" + assetType
	found = assetIsActive(stub, sAssetKey)
	if !found {
		err := fmt.Errorf("readAssetHistory arg asset %s does not exist", assetID)
		log.Error(err)
		return nil, err
	}

	// Get the history from the ledger
	stateHistory, err := readStateHistory(stub, sAssetKey)
	if err != nil {
		err = fmt.Errorf("readAssetHistory assetID %s of type %s failed readStateHistory: %s", assetID, assetType, err)
		log.Error(err)
		return nil, err
	}

	// is count present?
	var olen int
	countBytes, found := getObject(argsMap, "count")
	if found {
		olen = int(countBytes.(float64))
	}
	if olen <= 0 || olen > len(stateHistory.AssetHistory) {
		olen = len(stateHistory.AssetHistory)
	}
	var hStatesOut = make([]interface{}, 0, olen)
	for i := 0; i < olen; i++ {
		var obj interface{}
		err = json.Unmarshal([]byte(stateHistory.AssetHistory[i]), &obj)
		if err != nil {
			log.Errorf("readAssetHistory JSON unmarshal of entry %d failed [%#v]", i, stateHistory.AssetHistory[i])
			return nil, err
		}
		hStatesOut = append(hStatesOut, obj)
	}
	assetBytes, err = json.Marshal(hStatesOut)
	if err != nil {
		log.Errorf("readAssetHistory failed to marshal results: %s", err)
		return nil, err
	}

	return []byte(assetBytes), nil
}

//***************************************************
//***************************************************
//* CONTRACT STATE
//***************************************************
//***************************************************

func (t *SimpleChaincode) readContractState(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	if len(args) != 0 {
		err = errors.New("Too many arguments. Expecting none.")
		log.Error(err)
		return nil, err
	}

	// Get the state from the ledger
	chaincodeBytes, err := stub.GetState(CONTRACTSTATEKEY)
	if err != nil {
		err = fmt.Errorf("readContractState failed GETSTATE: %s", err)
		log.Error(err)
		return nil, err
	}

	return chaincodeBytes, nil
}

//***************************************************
//***************************************************
//* CONTRACT METADATA / SCHEMA INTERFACE
//***************************************************
//***************************************************



// ************************************
// readContractObjectModel
// ************************************
func (t *SimpleChaincode) readContractObjectModel(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var state = ContractState{MYVERSION, DEFAULTNICKNAME, make(map[string]bool),make(map[string]bool)}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		err := fmt.Errorf("JSON Marshal failed for get contract object model empty state: %+v with error [%s]", state, err)
		log.Error(err)
		return nil, err
	}
	return stateJSON, nil
}

// ************************************
// setLoggingLevel
// ************************************
func (t *SimpleChaincode) setLoggingLevel(stub shim.ChaincodeStubInterface, args []string) error {
	type LogLevelArg struct {
		Level string `json:"logLevel"`
	}
	var level LogLevelArg
	var err error
	if len(args) != 1 {
		err = errors.New("Incorrect number of arguments. Expecting a JSON encoded LogLevel.")
		log.Error(err)
		return err
	}
	err = json.Unmarshal([]byte(args[0]), &level)
	if err != nil {
		err = fmt.Errorf("setLoggingLevel failed to unmarshal arg: %s", err)
		log.Error(err)
		return err
	}
	for i, lev := range logLevelNames {
		if strings.ToUpper(level.Level) == lev {
			(*log).SetLoggingLevel(LogLevel(i))
			return nil
		}
	}
	err = fmt.Errorf("Unknown Logging level: %s", level.Level)
	log.Error(err)
	return err
}

// CreateOnUpdate is a shared parameter structure for the use of
// the createonupdate feature
type CreateOnUpdate struct {
	CreateOnUpdate bool `json:"createOnUpdate"`
}

// ************************************
// setCreateOnUpdate
// ************************************
func (t *SimpleChaincode) setCreateOnUpdate(stub shim.ChaincodeStubInterface, args []string) error {
	var createOnUpdate CreateOnUpdate
	var err error
	if len(args) != 1 {
		err = errors.New("setCreateOnUpdate expects a single parameter")
		log.Error(err)
		return err
	}
	err = json.Unmarshal([]byte(args[0]), &createOnUpdate)
	if err != nil {
		err = fmt.Errorf("setCreateOnUpdate failed to unmarshal arg: %s", err)
		log.Error(err)
		return err
	}
	err = PUTcreateOnUpdate(stub, createOnUpdate)
	if err != nil {
		err = fmt.Errorf("setCreateOnUpdate failed to PUT setting: %s", err)
		log.Error(err)
		return err
	}
	return nil
}

// PUTcreateOnUpdate marshals the new setting and writes it to the ledger
func PUTcreateOnUpdate(stub shim.ChaincodeStubInterface, createOnUpdate CreateOnUpdate) (err error) {
	createOnUpdateBytes, err := json.Marshal(createOnUpdate)
	if err != nil {
		err = errors.New("PUTcreateOnUpdate failed to marshal")
		log.Error(err)
		return err
	}
	err = stub.PutState("CreateOnUpdate", createOnUpdateBytes)
	if err != nil {
		err = fmt.Errorf("PUTSTATE createOnUpdate failed: %s", err)
		log.Error(err)
		return err
	}
	return nil
}

// canCreateOnUpdate retrieves the setting from the ledger and returns it to the calling function
func canCreateOnUpdate(stub shim.ChaincodeStubInterface) bool {
	var createOnUpdate CreateOnUpdate
	createOnUpdateBytes, err := stub.GetState("CreateOnUpdate")
	if err != nil {
		err = fmt.Errorf("GETSTATE for canCreateOnUpdate failed: %s", err)
		log.Error(err)
		return true // true is the default
	}
	err = json.Unmarshal(createOnUpdateBytes, &createOnUpdate)
	if err != nil {
		err = fmt.Errorf("canCreateOnUpdate failed to marshal: %s", err)
		log.Error(err)
		return true // true is the default
	}
	return createOnUpdate.CreateOnUpdate
}
// *********************************** ContractState ***************************************************************

// GETContractStateFromLedger retrieves state from ledger and returns to caller
func GETContractStateFromLedger(stub shim.ChaincodeStubInterface) (ContractState, error) {
    var state = ContractState{ MYVERSION, DEFAULTNICKNAME, make(map[string]bool),make(map[string]bool) }
    var err error
	contractStateBytes, err := stub.GetState(CONTRACTSTATEKEY)
    // minimum string is {"version":""} and version cannot be empty 
	if err == nil && len(contractStateBytes) > 14 {    
		// apparently, this blockchain instance is being reloaded, has the version changed?
		err = json.Unmarshal(contractStateBytes, &state)
		if err != nil {
            err = fmt.Errorf("Unmarshal failed for contract state: %s", err)
            log.Critical(err)
			return ContractState{}, err
		}
        if MYVERSION != state.Version {
            log.Noticef("Contract version has changed from %s to %s", state.Version, MYVERSION)
            state.Version = MYVERSION
        }
	} else {
        // empty state already initialized 
		log.Noticef("Initialized newly deployed contract state version %s", state.Version)
	}
    // this MUST be here
    if state.ActiveAssets == nil {
        state.ActiveAssets = make(map[string]bool)
    }
	 if state.ActiveAccounts == nil {
        state.ActiveAccounts = make(map[string]bool)
    }
    log.Debug("GETContractState successful")
    return state, nil 
}

// PUTContractStateToLedger writes a contract state into the ledger
func PUTContractStateToLedger(stub shim.ChaincodeStubInterface, state ContractState) (error) {
    var contractStateJSON []byte
    var err error
    contractStateJSON, err = json.Marshal(state)
    if err != nil {
        err = fmt.Errorf("Failed to marshal contract state: %s", err)
        log.Critical(err)
        return err
    }
    err = stub.PutState(CONTRACTSTATEKEY, contractStateJSON)
    if err != nil {
        err = fmt.Errorf("Failed to PUTSTATE contract state: %s", err)
        log.Critical(err)
        return err
    } 
    log.Debugf("PUTContractState: %#v", state)
    return nil 
}

func addAssetToContractState(stub shim.ChaincodeStubInterface, sAssetKey string) (error) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)  
    if err != nil {
        return err
    }
    log.Debugf("Adding asset %s to contract", sAssetKey)
    state.ActiveAssets[sAssetKey] = true
    return PUTContractStateToLedger(stub, state)
}

func removeAssetFromContractState(stub shim.ChaincodeStubInterface, assetID string) (error) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)  
    if err != nil {
        return err
    }
    log.Debugf("Deleting asset %s from contract", assetID)
    delete(state.ActiveAssets, assetID)
    return PUTContractStateToLedger(stub, state)
}

func getActiveAssets(stub shim.ChaincodeStubInterface) ([]string, error) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)  
    if err != nil {
        return []string{}, err
    }
    var a = make([]string, len(state.ActiveAssets))
    i := 0
    for id := range state.ActiveAssets {
        a[i] = id
        i++ 
    }
    sort.Strings(a)
    return a, nil
}

func initializeContractState(stub shim.ChaincodeStubInterface, version string, nickname string) (error) {
    var state ContractState
    var err error
    if version != MYVERSION {
        err = fmt.Errorf("Contract version: %s does not match version argument: %s", MYVERSION, version)
        log.Critical(err)
        return err
    }
    state, err = GETContractStateFromLedger(stub)
    if err != nil {
        return err
    }  
    if version != state.Version {
        log.Noticef("Contract version has changed from %s to %s", version, MYVERSION)
        // keep going, this is an update of version -- later this will
        // be handled by pulling state from the superseded contract version
    }
    state.Version = MYVERSION
    state.Nickname = nickname
    return PUTContractStateToLedger(stub, state)
}

func getLedgerContractVersion(stub shim.ChaincodeStubInterface) (string, error) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)  
    if err != nil {
        return "", err
    }
    return state.Version, nil   
}

func assetIsActive(stub shim.ChaincodeStubInterface, sAssetKey string) (bool) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)
    if err != nil { return false}
    found, _ := state.ActiveAssets[sAssetKey]
    return found
}                      
//********************************* Recent ***************************************************
// GETRecentStatesFromLedger returns the unmarshaled recent states
func GETRecentStatesFromLedger(stub shim.ChaincodeStubInterface) (RecentStates, error) {
    var state = RecentStates{make([]string, 0, MaxRecentStates)}
    var err error
	recentStatesBytes, err := stub.GetState(RECENTSTATESKEY)
	if err == nil { 
		err = json.Unmarshal(recentStatesBytes, &state.RecentStates)
		if err != nil {
            log.Noticef("Unmarshal failed for recent states: %s", err)
		}
	}
    // this MUST be here
    if state.RecentStates == nil || len(state.RecentStates) == 0 {
        state.RecentStates = make([]string, 0, MaxRecentStates)
    }
    log.Debugf("GETRecentStates returns: %#v", state)
    return state, nil 
}

// PUTRecentStatesToLedger marshals and writes the recent states
func PUTRecentStatesToLedger(stub shim.ChaincodeStubInterface, state RecentStates) (error) {
    var recentStatesJSON []byte
    var err error
    recentStatesJSON, err = json.Marshal(state.RecentStates)
    if err != nil {
        log.Criticalf("Failed to marshal recent states: %s", err)
        return err
    }
    err = stub.PutState(RECENTSTATESKEY, recentStatesJSON)
    if err != nil {
        log.Criticalf("Failed to PUTSTATE recent states: %s", err)
        return err
    } 
    log.Debugf("PUTRecentStates: %#v", state)
    return nil 
}

func clearRecentStates(stub shim.ChaincodeStubInterface) (error) {
    var rstates RecentStates
    rstates.RecentStates = make([]string, 0, MaxRecentStates)
    return PUTRecentStatesToLedger(stub, rstates)
}

func pushRecentState (stub shim.ChaincodeStubInterface, state string,isAsset string) (error) {
    var rstate RecentStates
    var err error
    var assetID string
    
    assetID, err = getAssetIDFromState(state,isAsset)
    if err != nil {
        return err
    }
    rstate, err = GETRecentStatesFromLedger(stub)
    if err != nil {
        return err
    }
    
    // shift slice to the right
    assetPosn, err := findAssetInRecent(assetID, rstate) 
    if err != nil {
        return err
    } else if assetPosn == -1 {
        // grow if not at capacity, since this one is new
        if len(rstate.RecentStates) < MaxRecentStates {
            rstate.RecentStates = rstate.RecentStates[0 : len(rstate.RecentStates)+1]
        }
        // shift it all since not found
        copy(rstate.RecentStates[1:], rstate.RecentStates[0:])
    } else {
        if len(rstate.RecentStates) > 1 {
            // shift over top of the same asset, can appear only once
            copy(rstate.RecentStates[1:], rstate.RecentStates[0:assetPosn])
        }
    }
    rstate.RecentStates[0] = state
    return PUTRecentStatesToLedger(stub, rstate)
}

// typically called when an asset is deleted
func removeAssetFromRecentState (stub shim.ChaincodeStubInterface, assetID string) (error) {
    var rstate RecentStates
    var err error
    rstate, err = GETRecentStatesFromLedger(stub)
    if err != nil {
        return err
    }
    assetPosn, err := findAssetInRecent(assetID, rstate)
    if err != nil {
        return err
    } else if assetPosn == -1 {
        // nothing to do
        return nil
    } else {
        if len(rstate.RecentStates) > 0 {
            // shift slice to the left to close the hole left by the asset
            copy(rstate.RecentStates[assetPosn:], rstate.RecentStates[assetPosn+1:])
        }
        if len(rstate.RecentStates) > 0 {
            rstate.RecentStates = rstate.RecentStates[0 : len(rstate.RecentStates)-1]
        }
    }
    return PUTRecentStatesToLedger(stub, rstate)
}

func getAssetIDFromState(state string,isAsset string) (string, error) {

	var err error
	if isAsset == "0" {
	     	var substate AssetIDT
    err = json.Unmarshal([]byte(state), &substate)
    if err != nil {
        log.Errorf("getAssetIDFromState state unmarshal to AssetID failed: %s", err)
        return "", err
    }
    if len(substate.ID) == 0 {
        err = errors.New("getAssetIDFromState substate.common.assetID is blank")
        log.Error(err)
        return "", err
    }
    	return substate.ID, nil 
	}else if isAsset == "1"	{ 
      	var substate AccountIDT
	
	err = json.Unmarshal([]byte(state), &substate)
    if err != nil {
        log.Errorf("getAssetIDFromState state unmarshal to AssetID failed: %s", err)
        return "", err
    }
    if len(substate.ID) == 0 {
        err = errors.New("getAssetIDFromState substate.common.assetID is blank")
        log.Error(err)
        return "", err
    }
   	return substate.ID, nil 
	}
   	return "blank", nil 
}

func findAssetInRecent (assetID string, rstate RecentStates) (int, error) {
    // returns -1 to signify not found (or error)
    var err error
    var substate AssetIDT
    for i := 0; i < len(rstate.RecentStates); i++ {
        err = json.Unmarshal([]byte(rstate.RecentStates[i]), &substate)
        if err != nil {
            log.Errorf("findAssetInRecent JSON unmarshal of entry %d failed [%#v]", i, rstate.RecentStates[i])
            return -1, err
        }
        if substate.ID == assetID {
        	log.Debugf("findAssetInRecent found assetID %s at position %d in recent states", assetID, i)
            return i, nil
        }
    }
    // not found
    log.Debugf("findAssetInRecent Did not find assetID %s in recent states", assetID)
    return -1, nil
}

func readRecentStates(stub shim.ChaincodeStubInterface) ([]byte, error) {
	var err error
    var rstate RecentStates
    var rstateOut = make([]interface{}, 0, MaxRecentStates) 

	// Get the recent states from the ledger
    rstate, err = GETRecentStatesFromLedger(stub)
    if err != nil {
        return nil, err
    }
    for i := 0; i < len(rstate.RecentStates); i++ {
        var obj interface{}
        err = json.Unmarshal([]byte(rstate.RecentStates[i]), &obj)
        if err != nil {
            log.Errorf("findAssetInRecent JSON unmarshal of entry %d failed [%#v]", i, rstate.RecentStates[i])
            return nil, err
        }
        rstateOut = append(rstateOut, obj)
    }
    rsBytes, err := json.Marshal(rstateOut)
    if err != nil {
        log.Errorf("readRecentStates JSON marshal of result failed [%#v]", rstate.RecentStates)
        return nil, err
    }
	return rsBytes, nil
}
//***************************************************Map**********************************

// finds an object by its qualified name, which looks like "location.latitude"
// as one example. Returns as map[string]interface{} 
func getObject (objIn interface{}, qname string) (interface{}, bool) {
    // return a copy of the selected object
    // handles full qualified name, starting at object's root
    obj, found := objIn.(map[string]interface{})
    if !found {
        obj, found = objIn.(ArgsMap)
        if !found {
            log.Errorf("getObject passed a non-map / non-ArgsMap: %#v", objIn)
            return nil, false
        }
    }
    obj = map[string]interface{}(obj)
    var returnObj interface{} = obj
    s := strings.Split(qname, ".")
    // crawl the levels, skipping the # root
    for i, v := range s {
        //fmt.Printf("Prop %d is: %s\n", i, v)
        if i+1 == len(s) {
            // last level, has to be here
            return findObjectByKey(returnObj, v)
        }
        returnObj, found = (returnObj.(map[string]interface{})[v]).(map[string]interface{})
        if !found {
            log.Debugf("getObject cannot find level: %s", v)
            return nil, false
        }
    }
    return nil, false
}

// this small function isolates the getting of the object in case
// sensitive or case insensitive modes because they are quite different
// we must not modify the destination key 

func findObjectByKey (objIn interface{}, key string) (interface{}, bool) {
    objMap, found := objIn.(map[string]interface{})
    if found {
        dstKey, found := findMatchingKey(objMap, key)
        if found {
            objOut, found := objMap[dstKey]
            if found { 
                return objOut, found 
            }
        }
    }
    return nil, false
}

// finds a key that matches the incoming key, very useful to remove the 
// complexity of switching case insensitivity because we always need
// the destination key to stay intact to avoid making copies of that
// substructure as we copy fields from the incoming structure 
func findMatchingKey (objIn interface{}, key string) (string, bool) {
    objMap, found := objIn.(map[string]interface{})
    if !found {
        // not a map, cannot proceed
        log.Warningf("findMatchingKey objIn is not a map shape %+v", objIn)
        return "", false
    }
    if CASESENSITIVEMODE {
        // we can just use the key directly
        _, found := objMap[key] 
        return key, found
    }
    // we must visit all keys and compare using tolower on each side
    for k := range objMap {
        if strings.ToLower(k) == strings.ToLower(key) {
            log.Debugf("findMatchingKey found match! %s %s", k, key)
            return k, true
        }
    }
    log.Warningf("findMatchingKey did not find key %s", key)
    return "", false
}

// in a contract, src is usually the incoming update event, 
// and dst is the existing state from the ledger 

func contains(arr interface{}, val interface{}) bool {
    switch t := arr.(type) {
        case []string:
            arr2 := arr.([]string)
            for _, v := range arr2 {
                if v == val {
                    return true
                }
            }
        case []int:
            arr2 := arr.([]int)
            for _, v := range arr2 {
                if v == val {
                    return true
                }
            }
        case []float64:
            arr2 := arr.([]float64)
            for _, v := range arr2 {
                if v == val {
                    return true
                }
            }
        case []interface{}:
            arr2 := arr.([]interface{})
            for _, v := range arr2 {
                switch tt := val.(type) {
                    case string:
                        if v.(string) == val.(string) { return true }
                    case int:
                        if v.(int) == val.(int) { return true }
                    case float64:
                        if v.(float64) == val.(float64) { return true }
                    case interface{}:
                        if v.(interface{}) == val.(interface{}) { return true }
                    default:
                        log.Errorf("contains passed array containing unknown type: %+v\n", tt);
                        return false
                }
            }
        default:
            log.Errorf("contains passed array of unknown type: %+v\n", t);
            return false
    }
    return false
}

// deep merge src into dst and return dst
func deepMerge(srcIn interface{}, dstIn interface{}) (map[string]interface{}){
    src, found := srcIn.(map[string]interface{})
    if !found {
        log.Criticalf("Deep Merge passed source map of type: %s", reflect.TypeOf(srcIn)) 
        return nil 
    }
    dst, found := dstIn.(map[string]interface{})
    if !found {
        log.Criticalf("Deep Merge passed dest map of type: %s", reflect.TypeOf(dstIn)) 
        return nil 
    }
    for k, v := range src {
        switch v.(type) {
            case map[string]interface{}:
                // don't try hoisting dstKey calculation
                dstKey, found := findMatchingKey(dst, k)
                if found {
                    dstChild, found := dst[dstKey].(map[string]interface{})
                    if found {
                        // recursive deepMerge into existing key
                        dst[dstKey] = deepMerge(v.(map[string]interface{}), dstChild)
                    } 
                } else {
                    // copy entire map to incoming key
                    dst[k] = v
                }
            case []interface{}:
                dstKey, found := findMatchingKey(dst, k)
                if found {
                    dstChild, found := dst[dstKey].([]interface{})
                    if found {
                        // union
                        for elem := range v.([]interface{}) {
                            if !contains(dstChild, elem) {
                                dstChild = append(dstChild, elem)
                            }
                        } 
                    }
                } else {
                    // copy
                    dst[k] = v
                }
            default:
                // copy discrete types 
                dstKey, found := findMatchingKey(dst, k)
                if found {
                    dst[dstKey] = v
                } else {
                    dst[k] = v
                }
        }
    }
    return dst
}

// returns a string that is nicely indented
// if json fails for some reason, returns the %#v representation
func prettyPrint(m interface{}) (string) {
    bytes, err := json.MarshalIndent(m, "", "  ")
    if err == nil {
        return string(bytes)
    }
    return fmt.Sprintf("%#v", m) 
}
//***************************************Logger*******************************
//var goLogger *logging.Logger

// NewContractLogger creates a logger for the contract to use
func NewContractLogger(module string, level LogLevel) (*ContractLogger) {
    l := &ContractLogger{module, level}
    l.SetLoggingLevel(level)
    l.setModule(module)
    return l
}

//SetLoggingLevel is used to change the logging level while the smart contract is running
func (cl *ContractLogger) SetLoggingLevel(level LogLevel) {
    if level < CRITICAL || level > DEBUG {
        cl.level = DEFAULTLOGGINGLEVEL
    } else {
        cl.level = level
    }
}

func (cl *ContractLogger) setModule(module string) {
    if module == "" { module = DEFAULTNICKNAME }
    module += "-" + MYVERSION
    (*cl).module = module
    //goLogger = logging.MustGetLogger(module)
}

//*************
// print logger
//*************

const pf string = "%s [%s] %.4s %s" 

func buildLogString(module string, level LogLevel, msg interface{}) (string) {
    var a = fmt.Sprint(msg)
    var t = time.Now().Format("2006/01/02 15:04:05") 
    return fmt.Sprintf(pf, t, module, logLevelNames[level], a) 
}

// Critical logs a message using CRITICAL as log level.
func (cl *ContractLogger) Critical(msg interface{}) {
    if CRITICAL > cl.level { return }
	logMessage(CRITICAL, buildLogString(cl.module, CRITICAL, msg))
}

// Criticalf logs a message using CRITICAL as log level.
func (cl *ContractLogger) Criticalf(format string, args ...interface{}) {
    if CRITICAL > cl.level { return }
	logMessage(CRITICAL, buildLogString(cl.module, CRITICAL, fmt.Sprintf(format, args)))
}

// Error logs a message using ERROR as log level.
func (cl *ContractLogger) Error(msg interface{}) {
    if ERROR > cl.level { return }
	logMessage(ERROR, buildLogString(cl.module, ERROR, msg))
}

// Errorf logs a message using ERROR as log level.
func (cl *ContractLogger) Errorf(format string, args ...interface{}) {
    if ERROR > cl.level { return }
	logMessage(ERROR, buildLogString(cl.module, ERROR, fmt.Sprintf(format, args)))
}

// Warning logs a message using WARNING as log level.
func (cl *ContractLogger) Warning(msg interface{}) {
    if WARNING > cl.level { return }
	logMessage(WARNING, buildLogString(cl.module, WARNING, msg))
}

// Warningf logs a message using WARNING as log level.
func (cl *ContractLogger) Warningf(format string, args ...interface{}) {
    if WARNING > cl.level { return }
	logMessage(WARNING, buildLogString(cl.module, WARNING, fmt.Sprintf(format, args)))
}

// Notice logs a message using NOTICE as log level.
func (cl *ContractLogger) Notice(msg interface{}) {
    if NOTICE > cl.level { return }
	logMessage(NOTICE, buildLogString(cl.module, NOTICE, msg))
}

// Noticef logs a message using NOTICE as log level.
func (cl *ContractLogger) Noticef(format string, args ...interface{}) {
    if NOTICE > cl.level { return }
	logMessage(NOTICE, buildLogString(cl.module, NOTICE, fmt.Sprintf(format, args)))
}

// Info logs a message using INFO as log level.
func (cl *ContractLogger) Info(msg interface{}) {
    if INFO > cl.level { return }
    logMessage(INFO, buildLogString(cl.module, INFO, msg))
}

// Infof logs a message using INFO as log level.
func (cl *ContractLogger) Infof(format string, args ...interface{}) {
    if INFO > cl.level { return }
	logMessage(INFO, buildLogString(cl.module, INFO, fmt.Sprintf(format, args)))
}

// Debug logs a message using DEBUG as log level.
func (cl *ContractLogger) Debug(msg interface{}) {
    if DEBUG > cl.level { return }
	logMessage(DEBUG, buildLogString(cl.module, DEBUG, msg))
}

// Debugf logs a message using DEBUG as log level.
func (cl *ContractLogger) Debugf(format string, args ...interface{}) {
    if DEBUG > cl.level { return }
	logMessage(DEBUG, buildLogString(cl.module, DEBUG, fmt.Sprintf(format, args)))
}

func logMessage(ll LogLevel, msg string) {
    if !strings.HasSuffix(msg, "\n") {
        msg += "\n"
    }
    fmt.Print(msg)
/*
    removing logger dependency as it is quite literally the only include difference from 3.0.3 to 3.0.4
    // for logger, time is added on front, so delete date and time from
    // our messgage ... space separated
    msg = strings.SplitN(msg, " ", 3)[2] 
    switch ll {
        case CRITICAL :
            goLogger.Critical(msg)            
        case ERROR :
            goLogger.Error(msg)            
        case WARNING :
            goLogger.Warning(msg)            
        case NOTICE :
            goLogger.Notice(msg)            
        case INFO :
            goLogger.Info(msg)            
        case DEBUG :
            goLogger.Debug(msg)            
    }
 */
}

///**********************************************************assetHistroy
const STATEHISTORYKEY string = ".StateHistory"

type AssetStateHistory struct {
	AssetHistory []string `json:"assetHistory"`
}

// Create a new history entry in the ledger for an asset.,\
func createStateHistory(stub shim.ChaincodeStubInterface, assetID string, stateJSON string) error {

	var ledgerKey = assetID + STATEHISTORYKEY
	var assetStateHistory = AssetStateHistory{make([]string, 1)}
	assetStateHistory.AssetHistory[0] = stateJSON

	assetState, err := json.Marshal(&assetStateHistory)
	if err != nil {
		return err
	}

	return stub.PutState(ledgerKey, []byte(assetState))

}

// Update the ledger with new state history for an asset. States are stored in the ledger in descending order by timestamp.
func updateStateHistory(stub shim.ChaincodeStubInterface, assetID string, stateJSON string) error {

	var ledgerKey = assetID + STATEHISTORYKEY
	var historyBytes []byte
	var assetStateHistory AssetStateHistory
	
	historyBytes, err := stub.GetState(ledgerKey)
	if err != nil {
		return err
	}

	err = json.Unmarshal(historyBytes, &assetStateHistory)
	if err != nil {
		return err
	}

	var newSlice []string = make([]string, 0)
	newSlice = append(newSlice, stateJSON)
	newSlice = append(newSlice, assetStateHistory.AssetHistory...)
	assetStateHistory.AssetHistory = newSlice

	assetState, err := json.Marshal(&assetStateHistory)
	if err != nil {
		return err
	}

	return stub.PutState(ledgerKey, []byte(assetState))

}

// Delete an state history from the ledger.
func deleteStateHistory(stub shim.ChaincodeStubInterface, assetID string) error {

	var ledgerKey = assetID + STATEHISTORYKEY
	return stub.DelState(ledgerKey)

}

// Get the state history for an asset.
func readStateHistory(stub shim.ChaincodeStubInterface, assetID string) (AssetStateHistory, error) {

	var ledgerKey = assetID + STATEHISTORYKEY
	var assetStateHistory AssetStateHistory
	var historyBytes []byte

	historyBytes, err := stub.GetState(ledgerKey)
	if err != nil {
		return assetStateHistory, err
	}

	err = json.Unmarshal(historyBytes, &assetStateHistory)
	if err != nil {
		return assetStateHistory, err
	}

	return assetStateHistory, nil

}
//*************************************Alert ***************
// Alerts exists so that strict type checking can be applied
type Alerts int32

const (
    // AlertsOVERTEMP the over temperature alert 
   // AlertsINVALIDCRTIME     Alerts = 0
    //AlertsINVALIDMDTIME     Alerts = 1
    AlertsTIMEERROR         Alerts = 0
    AlertsRPMERROR          Alerts = 1



    // AlertsSIZE is to be maintained always as 1 greater than the last alert, giving a size  
	AlertsSIZE        Alerts = 2
)

// AlertsName is a map of ID to name
var AlertsName = map[int]string{
	//0: "INVALID_CREATE_TIME",
   // 1: "INVALID_MODIFY_TIME",
    0: "CREATE_TIME_GREATER_THAN_MODIFY_TIME",
    1: "RPM_LESS_THAN_20PERCENT",
}

// AlertsValue is a map of name to ID
var AlertsValue = map[string]int32{
	//"INVALID_CREATE_TIME": 0,
    //"INVALID_MODIFY_TIME": 1,
    "CREATE_TIME_GREATER_THAN_MODIFY_TIME":0,
    "RPM_LESS_THAN_20PERCENT":1,
}

func (x Alerts) String() string {
	return AlertsName[int(x)]
}

// AlertArrayInternal is used to store the list of active, raised or cleared alerts
// for internal processing
type AlertArrayInternal [AlertsSIZE]bool
// AlertNameArray is used for external alerts in JSON
type AlertNameArray []string

// NOALERTSACTIVEINTERNAL is the zero value of an internal alerts array (bools)
var NOALERTSACTIVEINTERNAL = AlertArrayInternal{}
// NOALERTSACTIVE is the zero value of an external alerts array (string names)
var NOALERTSACTIVE = AlertNameArray{}

// AlertStatusInternal contains the three possible statuses for alerts
type AlertStatusInternal struct {
    Active  AlertArrayInternal  
    Raised  AlertArrayInternal  
    Cleared AlertArrayInternal  
}

type AlertStatus struct {
    Active  AlertNameArray  `json:"active"`
    Raised  AlertNameArray  `json:"raised"`
    Cleared AlertNameArray  `json:"cleared"`
}

// convert from external representation with slice of names
// to full length array of bools 
func (a *AlertStatus) asAlertStatusInternal() (AlertStatusInternal) {
    var aOut = AlertStatusInternal{}
    for i := range a.Active {
        aOut.Active[AlertsValue[a.Active[i]]] = true
    }
    for i := range a.Raised {
        aOut.Raised[AlertsValue[a.Raised[i]]] = true
    }
    for i := range a.Cleared {
        aOut.Cleared[AlertsValue[a.Cleared[i]]] = true
    }
    return aOut
}

// convert from internal representation with full length array of bools  
// to slice of names
func (a *AlertStatusInternal) asAlertStatus() (AlertStatus) {
    var aOut = newAlertStatus()
    for i := range a.Active {
        if a.Active[i] {
            aOut.Active = append(aOut.Active, AlertsName[i])
        }
    }
    for i := range a.Raised {
        if a.Raised[i] {
            aOut.Raised = append(aOut.Raised, AlertsName[i])
        }
    }
    for i := range a.Cleared {
        if a.Cleared[i] {
            aOut.Cleared = append(aOut.Cleared, AlertsName[i])
        }
    }
    return aOut
}

func (a *AlertStatusInternal) raiseAlert (alert Alerts) {
    if a.Active[alert] {
        // already raised
        // this is tricky, should not say this event raised an
        // active alarm, as it makes it much more difficult to track
        // the exact moments of transition
        a.Active[alert] = true
        a.Raised[alert] = false
        a.Cleared[alert] = false
    } else {
        // raising it
        a.Active[alert] = true
        a.Raised[alert] = true
        a.Cleared[alert] = false
    }
}

func (a *AlertStatusInternal) clearAlert (alert Alerts) {
    if a.Active[alert] {
        // clearing alert
        a.Active[alert] = false
        a.Raised[alert] = false
        a.Cleared[alert] = true
    } else {
        // was not active
        a.Active[alert] = false
        a.Raised[alert] = false
        // this is tricky, should not say this event cleared an
        // inactive alarm, as it makes it much more difficult to track
        //  the exact moments of transition
        a.Cleared[alert] = false
    }
}

func newAlertStatus() (AlertStatus) {
    var a AlertStatus
    a.Active = make([]string, 0, AlertsSIZE)
    a.Raised = make([]string, 0, AlertsSIZE)
    a.Cleared = make([]string, 0, AlertsSIZE)
    return a
}

func (a *AlertStatus) alertStatusFromMap (aMap map[string]interface{}) () {
    a.Active.copyFrom(aMap["active"].([]interface{}))
    a.Raised.copyFrom(aMap["raised"].([]interface{}))
    a.Cleared.copyFrom(aMap["cleared"].([]interface{}))
} 

func (arr *AlertNameArray) copyFrom (s []interface{}) {
    // a conversion like this must assert type at every level
    for i := 0; i < len(s); i++ {
        *arr = append(*arr, s[i].(string))
    }
}

// NoAlertsActive returns true when no alerts are active in the asset's status at this time
func (arr *AlertStatusInternal) NoAlertsActive() (bool) {
    return (arr.Active == NOALERTSACTIVEINTERNAL)
}

// AllClear returns true when no alerts are active, raised or cleared in the asset's status at this time
func (arr *AlertStatusInternal) AllClear() (bool) {
    return  (arr.Active == NOALERTSACTIVEINTERNAL) &&
            (arr.Raised == NOALERTSACTIVEINTERNAL) &&
            (arr.Cleared == NOALERTSACTIVEINTERNAL) 
}

// NoAlertsActive returns true when no alerts are active in the asset's status at this time
func (a *AlertStatus) NoAlertsActive() (bool) {
    return len(a.Active) == 0
}

// AllClear returns true when no alerts are active, raised or cleared in the asset's status at this time
func (a *AlertStatus) AllClear() (bool) {
    return  len(a.Active) == 0 &&
            len(a.Raised) == 0 &&
            len(a.Cleared) == 0 
}

//// Executing

func (a *ArgsMap) executeRules(alerts *AlertStatus) (bool) {
    log.Debugf("Executing rules input: %v", *alerts)
    var internal = (*alerts).asAlertStatusInternal()

    // rule 1 -- Create and mod time check
    internal.timeCheck(a)
    // rule 2 --RPM check : if motor is running at 20% or below, it will likely overheat
    internal.rpmCheck(a)
    // rule 3 -- HVAC Check. If the HVAC is not running, that is an alert scenario
    //internal.hvacCheck(a)
    // now transform internal back to external in order to give the contract the
    // appropriate JSON to send externally
    *alerts = internal.asAlertStatus()
    log.Debugf("Executing rules output: %v", *alerts)

    // set compliance true means out of compliance
    compliant := internal.calculateContractCompliance(a)
    // returns true if anything at all is active (i.e. NOT compliant)
    return !compliant
}

//***********************************
//**           RULES               **
//***********************************

func (alerts *AlertStatusInternal) timeCheck (a *ArgsMap) {
//var createTime time.Time
//var modTime time.Time
    /*
    now := time.Now()
    unixNano := now.UnixNano()                                                                      
    umillisec := unixNano / 1000000  */
    crTime, found := getObject(*a, "create_date")
    mdTime, found2 := getObject(*a, "last_mod_date")
    if found && found2 {
        //modTime= time.Unix(0, msInt*int64(time.Millisecond))
        if crTime.(float64) > mdTime.(float64) {
            alerts.raiseAlert(AlertsTIMEERROR)
        return
        }
        alerts.clearAlert(AlertsTIMEERROR)
    }
}
// Need to modify so that for motor, this ic called first 
func (alerts *AlertStatusInternal) rpmCheck (a *ArgsMap) {
//Reference : http://www.vfds.in/be-aware-of-vfd-running-in-low-speed-frequency-655982.html
    maxRPM, found := getObject(*a, "max_rpm")
    if found {
        curRPM, found2 := getObject(*a, "rpm")
        if found2 {
            percRPM := (curRPM.(float64)/maxRPM.(float64))*100
            if percRPM <=30 {
                alerts.raiseAlert(AlertsRPMERROR)
                return
            }
        }
    }
    alerts.clearAlert(AlertsRPMERROR)
}
/*
func (alerts *AlertStatusInternal) hvacCheck (a *ArgsMap) {
    hvacMode, found := getObject(*a, "hvac_mode")
    if found {
        tgtTemp, found2 := getObject(*a, "target_temperature_c")
        if found2 {
            ambTemp, found3 := getObject(*a, "ambient_temperature_c")
            if found3 {
                if (ambTemp.(float64) >tgtTemp.(float64) && hvacMode =="heat") {
                    alerts.raiseAlert(AlertsHVACOVERHEAT)
                    return
                }
                alerts.clearAlert(AlertsHVACOVERHEAT)
                if (ambTemp.(float64) <tgtTemp.(float64) && hvacMode =="cool") {
                    alerts.raiseAlert(AlertsHVACOVERCOOL)
                    return
                }
                alerts.clearAlert(AlertsHVACOVERCOOL)
            }
        }
    }
    alerts.clearAlert(AlertsHVACOVERHEAT)
    alerts.clearAlert(AlertsHVACOVERCOOL)
}
*/
//***********************************
//**         COMPLIANCE            **
//***********************************

func (alerts *AlertStatusInternal) calculateContractCompliance (a *ArgsMap) (bool) {
    // a simplistic calculation for this particular contract, but has access
    // to the entire state object and can thus have at it
    // compliant is no alerts active
    return alerts.NoAlertsActive()
    // NOTE: There could still a "cleared" alert, so don't go
    //       deleting the alerts from the ledger just on this status.
}
//****************************************Create Accout**************************************************

// ************************************
// createAccount
// ************************************
func (t *SimpleChaincode) createAccount(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var accountID string
	var assetType string
	var accountName string
	var argsMap ArgsMap
	var event interface{}
	var found bool
	var err error
	//var timeIn time.Time

	log.Info("Entering createAsset")

	// allowing 2 args because updateAsset is allowed to redirect when
	// asset does not exist
	if len(args) < 1 || len(args) > 2 {
		err = errors.New("Expecting one JSON event object")
		log.Error(err)
		return nil, err
	}

	accountID = ""
	
	accountName = ""
	eventBytes := []byte(args[0])
	log.Debugf("createAccount arg: %s", args[0])
	fmt.Println("args[0]",args[0])
	err = json.Unmarshal(eventBytes, &event)
	if err != nil {
		log.Errorf("createAccount failed to unmarshal arg: %s", err)
		return nil, err
	}

	if event == nil {
		err = errors.New("createAccount unmarshal arg created nil event")
		log.Error(err)
		return nil, err
	}

	argsMap, found = event.(map[string]interface{})
	if !found {
		err := errors.New("createAccount arg is not a map shape")
		log.Error(err)
		return nil, err
	}

	// is accountID present or blank?
	assetIDBytes, found := getObject(argsMap, ACCOUNTID)
	fmt.Println("assetIDBytes",assetIDBytes)
	
	if found {
		accountID, found = assetIDBytes.(string)
		if !found || accountID == "" {
			err := errors.New("createAccount arg does not include accountID ")
			log.Error(err)
			return nil, err
		}
	}
	// Is asset name present?
	assetTypeBytes, found := getObject(argsMap, ACCOUNTNAME)
	if found {
		accountName, found = assetTypeBytes.(string)
		if !found || accountName == "" {
			err := errors.New("createAsset arg does not include accountName ")
			log.Error(err)
			return nil, err
		}
	}


    assetType="account"
	sAccountKey := accountID + "_" + assetType
	fmt.Println("sAccountKey",sAccountKey)
	found = accountIsActive(stub, sAccountKey)
	if found {
		err := fmt.Errorf("createAsset arg asset %s already exists", accountID)
		log.Error(err)
		return nil, err
	}

	// For now, timestamp is being sent in from the invocation to the contract
	// Once the BlueMix instance supports GetTxnTimestamp, we will incorporate the
	// changes to the contract

	// run the rules and raise or clear alerts
	alerts := newAlertStatus()
	if argsMap.executeRules(&alerts) {
		// NOT compliant!
		log.Noticef("createAsset accountID %s is noncompliant", accountID)
		argsMap["alerts"] = alerts
		delete(argsMap, "incompliance")
	} else {
		if alerts.AllClear() {
			// all false, no need to appear
			delete(argsMap, "alerts")
		} else {
			argsMap["alerts"] = alerts
		}
		argsMap["incompliance"] = true
	}

	// copy incoming event to outgoing state
	// this contract respects the fact that createAsset can accept a partial state
	// as the moral equivalent of one or more discrete events
	// further: this contract understands that its schema has two discrete objects
	// that are meant to be used to send events: common, and custom
	stateOut := argsMap

	// save the original event
	stateOut["lastEvent"] = make(map[string]interface{})
	stateOut["lastEvent"].(map[string]interface{})["function"] = "createAccount"
	stateOut["lastEvent"].(map[string]interface{})["args"] = args[0]
	if len(args) == 2 {
		// in-band protocol for redirect
		stateOut["lastEvent"].(map[string]interface{})["redirectedFromFunction"] = args[1]
	}

	// marshal to JSON and write
	stateJSON, err := json.Marshal(&stateOut)
	if err != nil {
		err := fmt.Errorf("createAccount state for accountID %s failed to marshal", accountID)
		log.Error(err)
		return nil, err
	}

	// finally, put the new state
	log.Infof("Putting new account state %s to ledger", string(stateJSON))
	// The key i 'assetid'_'type'

	err = stub.PutState(sAccountKey, []byte(stateJSON))
	if err != nil {
		err = fmt.Errorf("createAccount accountID %s PUTSTATE failed: %s", accountID, err)
		log.Error(err)
		return nil, err
	}
	log.Infof("createAccount accountID  state %s successfully written to ledger: %s", accountID,  string(stateJSON))

	// add asset to contract state
	err = addAccountToContractState(stub, sAccountKey)
	if err != nil {
		err := fmt.Errorf("createAccount asset %s  failed to write asset state: %s", accountID,  err)
		log.Critical(err)
		return nil, err
	}
fmt.Println("stateJSON",stateJSON)
	err = pushRecentState(stub, string(stateJSON),"1")
	if err != nil {
		err = fmt.Errorf("createAccount accountID %s  push to recentstates failed: %s", accountID,  err)
		log.Error(err)
		return nil, err
	}

	// save state history
	err = createStateHistory(stub, sAccountKey, string(stateJSON))
	if err != nil {
		err := fmt.Errorf("createAccount asset %s of type %s state history save failed: %s", accountID, sAccountKey, err)
		log.Critical(err)
		return nil, err
	}
	return nil, nil
}

func accountIsActive(stub shim.ChaincodeStubInterface, sAssetKey string) (bool) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)
    if err != nil { return false}
    found, _ := state.ActiveAccounts[sAssetKey]
    return found
}

func addAccountToContractState(stub shim.ChaincodeStubInterface, sAssetKey string) (error) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)  
    if err != nil {
        return err
    }
    log.Debugf("Adding asset %s to contract", sAssetKey)
    state.ActiveAccounts[sAssetKey] = true
    return PUTContractStateToLedger(stub, state)
}// ************************************
// readAllAssets
// ************************************
func (t *SimpleChaincode) readAllAccounts(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var sAssetKey string
	var err error
	var results []interface{}
	var state interface{}

	if len(args) > 0 {
		err = errors.New("readAllAccounts expects no arguments")
		log.Error(err)
		return nil, err
	}

	aa, err := getActiveAccounts(stub)
	fmt.Println("aa",aa)
	if err != nil {
		err = fmt.Errorf("readAllAccounts failed to get the active assets: %s", err)
		log.Error(err)
		return nil, err
	}
	results = make([]interface{}, 0, len(aa))
	for i := range aa {
		sAssetKey = aa[i]
		// Get the state from the ledger
		assetBytes, err := stub.GetState(sAssetKey)
		if err != nil {
			// best efforts, return what we can
			log.Errorf("readAllAccounts assetID %s failed GETSTATE", sAssetKey)
			continue
		} else {
			err = json.Unmarshal(assetBytes, &state)
			if err != nil {
				// best efforts, return what we can
				log.Errorf("readAllAccounts assetID %s failed to unmarshal", sAssetKey)
				continue
			}
			results = append(results, state)
		}
	}

	resultsStr, err := json.Marshal(results)
	if err != nil {
		err = fmt.Errorf("readAllAccounts failed to marshal results: %s", err)
		log.Error(err)
		return nil, err
	}

	return []byte(resultsStr), nil
}

func getActiveAccounts(stub shim.ChaincodeStubInterface) ([]string, error) {
    var state ContractState
    var err error
    state, err = GETContractStateFromLedger(stub)  
    if err != nil {
        return []string{}, err
    }
    var a = make([]string, len(state.ActiveAccounts))
    i := 0
    for id := range state.ActiveAccounts {
        a[i] = id
        i++ 
    }
    sort.Strings(a)
    return a, nil
}


