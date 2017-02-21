package main

// "encoding/json"
//     "errors"
//     "reflect"

import (    
    "fmt"         
    "github.com/hyperledger/fabric/core/chaincode/shim"
)


type SimpleChaincode struct {
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
    return nil, nil
}

func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    // Handle different functions
    // if function == "createAsset" {
    //     // create assetID
    //     return t.createAsset(stub, args)
    // } else if function == "updateAsset" {
    //     // create assetID
    //     return t.updateAsset(stub, args)
    // } else if function == "deleteAsset" {
    //     // Deletes an asset by ID from the ledger
    //     return t.deleteAsset(stub, args)
    // }
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

   if function == "readAsset" {
        // gets the state for an assetID as a JSON struct
        return t.readTest(stub, args)
    }

    return nil, nil
}


func (t *SimpleChaincode) readTest(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    // var assetID string // asset ID
    // var err error
    // var state AssetState

    //  // validate input data for number of args, Unmarshaling to asset state and obtain asset id
    // stateIn, err:= t.validateInput(args)
    // if err != nil {
    //     return nil, errors.New("Asset does not exist!")
    // }
    // assetID = *stateIn.AssetID
    //     // Get the state from the ledger
    // assetBytes, err:= stub.GetState(assetID)
    // if err != nil  || len(assetBytes) ==0{
    //     err = errors.New("Unable to get asset state from ledger")
    //     return nil, err
    // } 
    // err = json.Unmarshal(assetBytes, &state)
    // if err != nil {
    //      err = errors.New("Unable to unmarshal state data obtained from ledger")
    //     return nil, err
    // }
    return nil, nil
}