/*******************************************************************************
Copyright (c) 2016 IBM Corporation and other Contributors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and limitations under the License.
Contributors:
Sumabala Nair - Initial Contribution
Kim Letkeman - Initial Contribution
Sumabala Nair - Updated for hyperledger May 2016
Sumabala Nair - Partial updates added May 2016
******************************************************************************/
//SN: March 2016

// IoT Blockchain Simple Smart Contract v 1.0

// This is a simple contract that creates a CRUD interface to 
// create, read, update and delete an asset

package main

import (
    "encoding/json"
    "errors"
    "crypto/sha1"
	"fmt"
	"encoding/binary"
	"bytes"
   "strings"
     "reflect"
     //"strconv"
    "github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

const CONTRACTSTATEKEY string = "ContractStateKey"  
// store contract state - only version in this example
const MYVERSION string = "1.0"

// ************************************
// asset and contract state 
// ************************************

type ContractState struct {
    Version      string                        `json:"version"`
}

type Geolocation struct {
    Latitude    *float64 `json:"latitude,omitempty"`
    Longitude   *float64 `json:"longitude,omitempty"`
}

type AssetState struct {
    AssetID        string       `json:"assetID,omitempty"`        // all assets must have an ID, primary key of contract
    AssetName      string       `json:"assetName,omitempty"`       // current asset location
    }

var contractState = ContractState{MYVERSION}
var listAsset [20]AssetState
// ************************************
// deploy callback mode 
// ************************************
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    var stateArg ContractState
    var err error
    /*var obj [5]AssetState
    fmt.Println(obj[0])
    for i := 0; i < 10; i++ {
		arrAssetState[0].AssetID = "i"
		arrAssetState[0].AssetName = "TUshar"	
	}	
	for j := 0; j < 10; j++ {
	fmt.Println(arrAssetState[0])		
	}*/
    fmt.Println("in init")    
    listAsset[0].AssetID = "1"    
    listAsset[0].AssetName = "a" 
    fmt.Println(listAsset[0])
    //listAsset[0] = obj 
    fmt.Println(listAsset)
    fmt.Println("after list")
    if len(args) != 1 {
        return nil, errors.New("init expects one argument, a JSON string with tagged version string")
    }
    err = json.Unmarshal([]byte(args[0]), &stateArg)
    if err != nil {
        return nil, errors.New("Version argument unmarshal failed: " + fmt.Sprint(err))
    }
    if stateArg.Version != MYVERSION {
        return nil, errors.New("Contract version " + MYVERSION + " must match version argument: " + stateArg.Version)
    }
    contractStateJSON, err := json.Marshal(stateArg)
    if err != nil {
        return nil, errors.New("Marshal failed for contract state" + fmt.Sprint(err))
    }
    err = stub.PutState(CONTRACTSTATEKEY, contractStateJSON)
    if err != nil {
        return nil, errors.New("Contract state failed PUT to ledger: " + fmt.Sprint(err))
    }
    return nil, nil
}

// ************************************
// deploy and invoke callback mode 
// ************************************
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    // Handle different functions
	fmt.Println("in invoke")
    fmt.Println(listAsset)
        return t.createAsset(stub, args)
    if function == "createAsset" {
        // create assetID
		fmt.Println("c1")
        return t.createAsset(stub, args)
    } else if function == "updateAsset" {
        // create assetID
			fmt.Println("c2")
        return t.updateAsset(stub, args)
    } else if function == "deleteAsset" {
        // Deletes an asset by ID from the ledger
			fmt.Println("c3")
        return t.deleteAsset(stub, args)
    }
	fmt.Println("c4")
    return nil, errors.New("Received unknown invocation: " + function)
}

// ************************************
// query callback mode 
// ************************************
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    // Handle different functions
    if function == "readAsset" {
        // gets the state for an assetID as a JSON struct
        return t.readAsset(stub, args)
    } else if function =="readAssetObjectModel" {
        return t.readAssetObjectModel(stub, args)
    }  
    return nil, errors.New("Received unknown invocation: " + function)
}

/**********main implementation *************/

func main() {
    err := shim.Start(new(SimpleChaincode))
    if err != nil {
        fmt.Printf("Error starting Simple Chaincode: %s", err)
    }
}

/*****************ASSET CRUD INTERFACE starts here************/

/****************** 'deploy' methods *****************/

/******************** createAsset ********************/

func (t *SimpleChaincode) createAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    _,erval:=t. createOrUpdateAsset(stub, args)
    return nil, erval
}

//******************** updateAsset ********************/

func (t *SimpleChaincode) updateAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
     _,erval:=t. createOrUpdateAsset(stub, args)
    return nil, erval
}


//******************** deleteAsset ********************/

func (t *SimpleChaincode) deleteAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    var assetID string // asset ID
    var err error
    var stateIn AssetState

    // validate input data for number of args, Unmarshaling to asset state and obtain asset id
    stateIn, err = t.validateInput(args)
    if err != nil {
        return nil, err
    }
    assetID = stateIn.AssetID
    // Delete the key / asset from the ledger
    err = stub.DelState(assetID)
    if err != nil {
        err = errors.New("DELSTATE failed! : "+ fmt.Sprint(err))
       return nil, err
    }
    return nil, nil
}

/******************* Query Methods ***************/

//********************readAsset********************/

func (t *SimpleChaincode) readAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    var assetID string // asset ID
    var err error
    var state AssetState
    fmt.Println("in readAsset")
     // validate input data for number of args, Unmarshaling to asset state and obtain asset id
    stateIn, err:= t.validateInput(args)
    fmt.Println("stateIn=",stateIn)
    if err != nil {
        return nil, errors.New("Asset does not exist!")
    }
    assetID = stateIn.AssetID
    fmt.Println("assetID=",assetID)
        // Get the state from the ledger
    assetBytes, err:= stub.GetState(assetID)
    fmt.Println("assetBytes=",assetBytes)
    if err != nil  || len(assetBytes) ==0{
        err = errors.New("Unable to get asset state from ledger")
        return nil, err
    } 
    err = json.Unmarshal(assetBytes, &stateIn)
    if err != nil {
         err = errors.New("Unable to unmarshal state data obtained from ledger")
        return nil, err
    }
    return assetBytes, nil
}

//*************readAssetObjectModel*****************/

func (t *SimpleChaincode) readAssetObjectModel(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    var state AssetState = AssetState{}

    // Marshal and return
    stateJSON, err := json.Marshal(state)
    if err != nil {
        return nil, err
    }
    return stateJSON, nil
}

// ************************************
// validate input data : common method called by the CRUD functions
// ************************************
func (t *SimpleChaincode) validateInput(args []string) (stateIn AssetState, err error) {
    //var assetID string // asset ID
    var state AssetState = AssetState{} // The calling function is expecting an object of type AssetState    	
    if len(args) !=1 {
        err = errors.New("Incorrect number of arguments. Expecting a JSON strings with mandatory assetID")
        return state, err
    }    
    jsonData:=args[0]
    var pro AssetState	
    err = json.NewDecoder(strings.NewReader(jsonData)).Decode(&pro)
    if err != nil {
	fmt.Println(err)
	return
    }
    //fmt.Println(pro.AssetID)
    //var i string
    //i = pro.AssetID // temporary start with AssetID = 1
    var index int
    index = 1 //strconv.Atoi(pro.AssetID)
    listAsset[index].AssetID = pro.AssetID
    listAsset[index].AssetName = pro.AssetName
    return pro, nil
    /*assetID = ""
    stateJSON := []byte(jsonData)    
    err = json.Unmarshal(stateJSON, &stateIn)
    if err != nil {
        err = errors.New("Unable to unmarshal input JSON data")
        return state, err        
    }      
    assetID = strings.TrimSpace(stateIn.AssetID)       
    stateIn.AssetID = assetID
    return stateIn, nil*/
}


//******************** createOrUpdateAsset ********************/

func (t *SimpleChaincode) createOrUpdateAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    var assetID string                 // asset ID                    // used when looking in map
    var err error
    var stateIn AssetState
    //var stateStub AssetState   
    //var x = []byte{}
    var bin_buf bytes.Buffer
    var buf []byte
    // validate input data for number of args, Unmarshaling to asset state and obtain asset id
	
    stateIn, err = t.validateInput(args)
    fmt.Println("createUpdate")
	assetID = stateIn.AssetID
    if err != nil {
        return nil, err
    }
	
    // Check if asset record existed in stub
    //assetBytes, err:= stub.GetState(assetID)
    //var length int
    //length = len(listAsset)
    //fmt.Println(length)
	//stateStub = stateIn
    //listAsset[len(listAsset)-1:][0]
    //fmt.Println("len of array" , listAsset[len(listAsset)-1])
	//fmt.Println("assetbyte= ", assetBytes)
    /*if err != nil || len(assetBytes)==0{
        // This implies that this is a 'create' scenario
         stateStub = stateIn // The record that goes into the stub is the one that cme in
    } */
    //stateJSON, err := json.Marshal(stateStub)
	/*for i:=0; i<len(listAsset); i++{
    b := []byte(listAsset[i])
    for j:=0; j<len(b); j++{
        x = append(x,b[j])
    }
    }*/    
	x := listAsset[1]
	binary.Write(&bin_buf, binary.BigEndian,x)
	fmt.Printf("% x", sha1.Sum(bin_buf.Bytes()))    
    buf, err = json.Marshal(bin_buf)
    //_, err = w.Write(buf)
    fmt.Println("buf",buf)
    err = stub.PutState(assetID, buf)
    if err != nil {
        err = errors.New("PUT ledger state failed: "+ fmt.Sprint(err))  
 	
        return nil, err
    } 
		
    return nil, nil
}
/*********************************  internal: mergePartialState ****************************/	
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