/*
 SPDX-License-Identifier: Apache-2.0
*/

// ====CHAINCODE EXECUTION SAMPLES (CLI) ==================

// ==== Invoke Thermal ====
// peer chaincode invoke -o orderer.example.com:7050 --tls --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem -C $CHANNEL_NAME -n thermal -c '{"Args":["initThermal","personid","location","time","temperature"]}'

// ==== Query marbles ====
// peer chaincode query -C $CHANNEL_NAME -n thermal -c '{"Args":["queryMarbles","{\"selector\":{\"keyword\":\"information\"}}"]}'
// peer chaincode query -C myc1 -n marbles -c '{"Args":["getMarblesByRange","marble1","marble3"]}'

// Rich Query with Pagination (Only supported if CouchDB is used as state database):
// peer chaincode query -C myc1 -n marbles -c '{"Args":["queryMarblesWithPagination","{\"selector\":{\"owner\":\"tom\"}}","3",""]}'

// INDEXES TO SUPPORT COUCHDB RICH QUERIES
//
// Indexes in CouchDB are required in order to make JSON queries efficient and are required for
// any JSON query with a sort. As of Hyperledger Fabric 1.1, indexes may be packaged alongside
// chaincode in a META-INF/statedb/couchdb/indexes directory. Each index must be defined in its own
// text file with extension *.json with the index definition formatted in JSON following the
// CouchDB index JSON syntax as documented at:
// http://docs.couchdb.org/en/2.1.1/api/database/find.html#db-index
//
// This marbles02 example chaincode demonstrates a packaged
// index which you can find in META-INF/statedb/couchdb/indexes/indexOwner.json.
// For deployment of chaincode to production environments, it is recommended
// to define any indexes alongside chaincode so that the chaincode and supporting indexes
// are deployed automatically as a unit, once the chaincode has been installed on a peer and
// instantiated on a channel. See Hyperledger Fabric documentation for more details.
//
// If you have access to the your peer's CouchDB state database in a development environment,
// you may want to iteratively test various indexes in support of your chaincode queries.  You
// can use the CouchDB Fauxton interface or a command line curl utility to create and update
// indexes. Then once you finalize an index, include the index definition alongside your
// chaincode in the META-INF/statedb/couchdb/indexes directory, for packaging and deployment
// to managed environments.
//
// In the examples below you can find index definitions that support marbles02
// chaincode queries, along with the syntax that you can use in development environments
// to create the indexes in the CouchDB Fauxton interface or a curl command line utility.
//

//Example hostname:port configurations to access CouchDB.
//
//To access CouchDB docker container from within another docker container or from vagrant environments:
// http://couchdb:5984/
//
//Inside couchdb docker container
// http://127.0.0.1:5984/

// Index for docType, owner.
//
// Example curl command line to define index in the CouchDB channel_chaincode database
// curl -i -X POST -H "Content-Type: application/json" -d "{\"index\":{\"fields\":[\"docType\",\"owner\"]},\"name\":\"indexOwner\",\"ddoc\":\"indexOwnerDoc\",\"type\":\"json\"}" http://hostname:port/myc1_marbles/_index
//

// peer chaincode query -C $CHANNEL_NAME -n thermal -c '{"Args":["queryThermal", "{\"selector\":{\"location\":\"IT2\"}}"]}'

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type thermal struct {
	ObjectType  string  `json:"docType"` //docType is used to distinguish the various types of objects in state database
	ID          string  `json:"id"`      //the fieldtags are needed to keep case from bouncing around
	Location    string  `json:"location"`
	Time        string  `json:"time"`
	Temperature float64 `json:"temperature"`
}

// ===================================================================================
// Main
// ===================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init initializes chaincode
// ===========================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	if function == "initThermal" { //create a new marble
		return t.initThermal(stub, args)
	} else if function == "delete" { //delete a marble
		return t.delete(stub, args)
	} else if function == "queryThermal" { //find marbles based on an ad hoc rich query
		return t.queryThermal(stub, args)
	} else if function == "getThermalByRange" {
		return t.getThermalByRange(stub, args)
	} else if function == "queryThermalByLocation" {
		return t.queryThermalByLocation(stub, args)
	} else if function == "queryThermalByTemperature" {
		return t.queryThermalByTemperature(stub, args)
	}

	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

// ============================================================
// initMarble - create a new marble, store into chaincode state
// ============================================================
func (t *SimpleChaincode) initThermal(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	//   0        1         2        3
	// "ID", "location", "time", "thermal"
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	// ==== Input sanitation ====
	fmt.Println("- start init thermal")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("4th argument must be a non-empty string")
	}
	personid := args[0]
	location := args[1]
	time := args[2]
	temperature, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}

	// ==== Create thermal object and marshal to JSON ====
	objectType := "thermal"
	thermal := &thermal{objectType, personid, location, time, temperature}
	thermalJSONasBytes, err := json.Marshal(thermal)
	if err != nil {
		return shim.Error(err.Error())
	}
	//Alternatively, build the marble json string manually if you don't want to use struct marshalling
	//marbleJSONasString := `{"docType":"Marble",  "name": "` + marbleName + `", "color": "` + color + `", "size": ` + strconv.Itoa(size) + `, "owner": "` + owner + `"}`
	//marbleJSONasBytes := []byte(str)

	// === Save marble to state ===
	err = stub.PutState(location+time, thermalJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	// ==== Marble saved and indexed. Return success ====
	fmt.Println("- end init thermal")
	return shim.Success(nil)
}

func (t *SimpleChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var thermalJSON thermal
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	personid := args[0]
	// to maintain the color~name index, we need to read the marble first and get its color
	valAsbytes, err := stub.GetState(personid) //get the marble from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + personid + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Person does not exist: " + personid + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &thermalJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + personid + "\"}"
		return shim.Error(jsonResp)
	}

	err = stub.DelState(personid) //remove the marble from chaincode state
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	return shim.Success(nil)
}

func (t *SimpleChaincode) queryThermal(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "queryString"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	queryString := args[0]

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func (t *SimpleChaincode) getThermalByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//      0       1       2
	//  location  time1   time2
	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	startKey := args[0] + args[1]
	endKey := args[0] + args[2]

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Printf("- getMarblesByRange queryResult:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func (t *SimpleChaincode) queryThermalByLocation(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	location := args[0]

	queryString := fmt.Sprintf("{\"selector\":{\"location\":\"%s\"}}", location)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func (t *SimpleChaincode) queryThermalByTemperature(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "temperature"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	temperature, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}

	queryString := fmt.Sprintf("{\"selector\":{\"temperature\":{\"$gt\":%f}}}", temperature)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return &buffer, nil
}
