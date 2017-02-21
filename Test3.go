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