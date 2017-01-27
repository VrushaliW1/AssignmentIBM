/*
Copyright IBM Corp 2016 All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		 http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"strconv"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
    var BalanceA, BalanceB int
    var AccountA, AccountB string
    var err error
	if len(args) != 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
    AccountA = args[0]
    BalanceA, err = strconv.Atoi(args[1])
     
    AccountB = args[2]
    BalanceB, err = strconv.Atoi(args[3])
	//err1 := stub.PutState("hello_world", []byte(args[0]))

	if err != nil {
		return nil, err
	}
    fmt.Printf("BalanceA = %d, BalanceB = %d\n", BalanceA, BalanceB)
    err = stub.PutState(AccountA, []byte(strconv.Itoa(BalanceA)))
    err = stub.PutState(AccountB, []byte(strconv.Itoa(BalanceB)))
	return nil, nil
}

// Invoke isur entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Printf("Running invoke")
	
	var AccountA, AccountB string    // Entities
	var BalanceA, BalanceB int // Asset holdings
	var X int          // Transaction value	

	AccountA = args[0]
	AccountB = args[1]

	// Get the state from the ledger
	// TODO: will be nice to have a GetAllState call to ledger
	Avalbytes, err := stub.GetState(AccountA)
	
	BalanceA, _ = strconv.Atoi(string(Avalbytes))

	Bvalbytes, err := stub.GetState(AccountB)
	
	BalanceB, _ = strconv.Atoi(string(Bvalbytes))

	// Perform the execution
	X, err = strconv.Atoi(args[2])
	BalanceA = BalanceA - X
	BalanceB = BalanceB + X
	fmt.Printf("Aval = %d, Bval = %d\n", BalanceA, BalanceB)

	err = stub.PutState(AccountA, []byte(strconv.Itoa(BalanceA)))
    err = stub.PutState(AccountB, []byte(strconv.Itoa(BalanceB)))
	if err != nil {
		return nil, err
	}
	return Avalbytes, nil
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	fmt.Println("query is running " + function)
    var Account string 
	var err error
    if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the person to query")
	}
    Account = args[0]
    Avalbytes, err := stub.GetState(Account)
    if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + Account + "\"}"
		return nil, errors.New(jsonResp)
	}

	if Avalbytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + Account + "\"}"
		return nil, errors.New(jsonResp)
	}

	jsonResp := "{\"Name\":\"" + Account + "\",\"Amount\":\"" + string(Avalbytes) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	
    return Avalbytes, nil
}

// write - invoke function to write key/value pair
func (t *SimpleChaincode) write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, value string
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the key and value to set")
	}

	key = args[0] //rename for funsies
	value = args[1]
	err = stub.PutState(key, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// read - query function to read key/value pair
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")
	}

	key = args[0]
	valAsbytes, err := stub.GetState(key)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil
}