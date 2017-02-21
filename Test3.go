package main

import (
    "strings"
    "encoding/json"
    "reflect"
    "errors"        
    "fmt"         
    "github.com/hyperledger/fabric/core/chaincode/shim"
)


type SimpleChaincode struct {
}

type AssetState struct {
    AssetID        *string       `json:"assetID,omitempty"`        // all assets must have an ID, primary key of contract
    //Location       *Geolocation  `json:"location,omitempty"`       // current asset location
    Temperature    *float64      `json:"temperature,omitempty"`    // asset temp
    Carrier        *string       `json:"carrier,omitempty"`        // the name of the carrier
}

func main() {
    err := shim.Start(new(SimpleChaincode))
    if err != nil {
        fmt.Printf("Error starting Simple Chaincode: %s", err)
    }
}

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    // var stateArg ContractState
    // var err error
    // if len(args) != 1 {
    //     return nil, errors.New("init expects one argument, a JSON string with tagged version string")
    // }
    // err = json.Unmarshal([]byte(args[0]), &stateArg)
    // if err != nil {
    //     return nil, errors.New("Version argument unmarshal failed: " + fmt.Sprint(err))
    // }
    // if stateArg.Version != MYVERSION {
    //     return nil, errors.New("Contract version " + MYVERSION + " must match version argument: " + stateArg.Version)
    // }
    // contractStateJSON, err := json.Marshal(stateArg)
    // if err != nil {
    //     return nil, errors.New("Marshal failed for contract state" + fmt.Sprint(err))
    // }
    // err = stub.PutState(CONTRACTSTATEKEY, contractStateJSON)
    // if err != nil {
    //     return nil, errors.New("Contract state failed PUT to ledger: " + fmt.Sprint(err))
    // }

    err := stub.PutState("AsID", []byte(args[0]))

    return nil, nil
}

func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    // Handle different functions
    if function == "createAsset" {
        // create assetID
        return t.createAsset(stub, args)
    // } else if function == "updateAsset" {
    //     // create assetID
    //     return t.updateAsset(stub, args)
    // } else if function == "deleteAsset" {
    //     // Deletes an asset by ID from the ledger
    //     return t.deleteAsset(stub, args)
    }
    return nil, nil
}


func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    // Handle different functions
    // if function == "readAsset" {
    //     // gets the state for an assetID as a JSON struct
    //     return t.readAsset(stub, args)
    // } else if function =="readAssetObjectModel" {
    //     return t.readAssetObjectModel(stub, args)
    // }  else if function == "readAssetSamples" {
	// 	// returns selected sample objects 
	// 	return t.readAssetSamples(stub, args)
	// } else if function == "readAssetSchemas" {
	// 	// returns selected sample objects 
	// 	return t.readAssetSchemas(stub, args)
	// }

    fmt.Println("In query")

   if function == "readAsset" {
        // gets the state for an assetID as a JSON struct
        return t.readTest(stub, args)
    }

    return nil, nil
}

func (t *SimpleChaincode) createAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    _,erval:=t. createOrUpdateAsset(stub, args)
    return nil, erval
}

func (t *SimpleChaincode) createOrUpdateAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    var assetID string                 // asset ID                    // used when looking in map
    var err error
    var stateIn AssetState
    var stateStub AssetState
   

    // validate input data for number of args, Unmarshaling to asset state and obtain asset id

    stateIn, err = t.validateInput(args)
    
    assetID = *stateIn.AssetID

    fmt.Println("assetID = " + string(assetID))

    assetBytes, err:= stub.GetState(assetID)    

    if err != nil || len(assetBytes)==0{
        fmt.Println("create 1 ")
        // This implies that this is a 'create' scenario
         stateStub = stateIn // The record that goes into the stub is the one that cme in         
    } 

    //var ID string = string(*stateStub.AssetID);

    stateJSON, err := json.Marshal(stateStub)
    
    err = stub.PutState(assetID, stateJSON)

    fmt.Println("stateJSON == " + string(stateJSON))

    fmt.Println("assetID == " + string(assetID))    

    return nil, nil
}

func (t *SimpleChaincode) validateInput(args []string) (stateIn AssetState, err error) {
    var assetID string // asset ID
    var state AssetState = AssetState{} // The calling function is expecting an object of type AssetState

    if len(args) !=1 {
        err = errors.New("Incorrect number of arguments. Expecting a JSON strings with mandatory assetID")
        return state, err
    }
    jsonData:=args[0]
    assetID = ""
    stateJSON := []byte(jsonData)
    err = json.Unmarshal(stateJSON, &stateIn)
    if err != nil {
        err = errors.New("Unable to unmarshal input JSON data")
        return state, err
        // state is an empty instance of asset state
    }      
    // was assetID present?
    // The nil check is required because the asset id is a pointer. 
    // If no value comes in from the json input string, the values are set to nil
    
    if stateIn.AssetID !=nil { 
        assetID = strings.TrimSpace(*stateIn.AssetID)
        if assetID==""{
            err = errors.New("AssetID not passed")
            return state, err
        }
    } else {
        err = errors.New("Asset id is mandatory in the input JSON data")
        return state, err
    }
    
    //fmt.Println("stateIn -- " + stateIn)
    
    stateIn.AssetID = &assetID
    return stateIn, nil
}

func (t *SimpleChaincode) readTest(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
     var assetID string // asset ID
     var err error
     var state AssetState

    //  // validate input data for number of args, Unmarshaling to asset state and obtain asset id
     stateIn, err:= t.validateInput(args)
     if err != nil {
         return nil, errors.New("Asset does not exist!")
     }
     assetID = string(*stateIn.AssetID)
     fmt.Println(assetID)
         // Get the state from the ledger
     assetBytes, err:= stub.GetState(assetID)
     fmt.Println(assetBytes)
     if err != nil  || len(assetBytes) ==0{
         err = errors.New("Unable to get asset state from ledger")
        return nil, err
     } 
     err = json.Unmarshal(assetBytes, &state)
     if err != nil {
          err = errors.New("Unable to unmarshal state data obtained from ledger")
         return nil, err
     }
    fmt.Println("In readTest")
     //var ID string = string(assetBytes);
    return assetBytes, nil
}



func (t *SimpleChaincode) mergePartialState(oldState AssetState, newState AssetState) (AssetState,  error) {
     
    old := reflect.ValueOf(&oldState).Elem()
    new := reflect.ValueOf(&newState).Elem()
    for i := 0; i < old.NumField(); i++ {
        oldOne:=old.Field(i)
        newOne:=new.Field(i)
        if ! reflect.ValueOf(newOne.Interface()).IsNil() {
            oldOne.Set(reflect.Value(newOne))
        } 
    }
    return oldState, nil
 }


