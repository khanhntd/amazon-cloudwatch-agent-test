// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/mitchellh/mapstructure"
)

type matrixRow struct {
	TestDir                 string `json:"test_dir"`
	Os                      string `json:"os"`
	Family                  string `json:"family"`
	TestType                string `json:"testType"`
	Arc                     string `json:"arc"`
	InstanceType            string `json:"instanceType"`
	Ami                     string `json:"ami"`
	BinaryName              string `json:"binaryName"`
	Username                string `json:"username"`
	InstallAgentCommand     string `json:"installAgentCommand"`
	CaCertPath              string `json:"caCertPath"`
	PerformanceNumberOfLogs string `json:"performance_number_of_logs"`
}

// you can't have a const map in golang
var testTypeToTestDirMap = map[string][]string{
	"ec2_gpu": {
		"./test/nvidia_gpu",
	},
	"ec2_linux": {
		"./test/ca_bundle",
		"./test/cloudwatchlogs",
		"./test/metrics_number_dimension",
		"./test/metric_value_benchmark",
		"./test/run_as_user",
		"./test/collection_interval",
		"./test/metric_dimension",
	},
	"ec2_performance": {
		"./test/performancetest",
	},
	"ecs_fargate": {
		"./test/ecs/ecs_metadata",
	},
	"ecs_ec2_daemon": {
		"./test/metric_value_benchmark",
	},
}

func main() {
	for testType, testDir := range testTypeToTestDirMap {
		testMatrix := genMatrix(testType, testDir)
		writeTestMatrixFile(testType, testMatrix)
	}
}

func genMatrix(testType string, testDirList []string) []matrixRow {
	openTestMatrix, err := os.Open(fmt.Sprintf("generator/resources/%v_test_matrix.json", testType))

	if err != nil {
		log.Panicf("can't read file %v_test_matrix.json err %v", testType, err)
	}

	byteValueTestMatrix, _ := ioutil.ReadAll(openTestMatrix)
	_ = openTestMatrix.Close()

	var testMatrix []map[string]string
	err = json.Unmarshal(byteValueTestMatrix, &testMatrix)
	if err != nil {
		log.Panicf("can't unmarshall file %v_test_matrix.json err %v", testType, err)
	}

	testMatrixComplete := make([]matrixRow, 0, len(testMatrix))
	for _, test := range testMatrix {
		for _, testDirectory := range testDirList {
			row := matrixRow{TestDir: testDirectory, TestType: testType}
			err = mapstructure.Decode(test, &row)
			if err != nil {
				log.Panicf("can't decode map test %v to metric line struct with error %v", testDirectory, err)
			}
			testMatrixComplete = append(testMatrixComplete, row)
		}
	}
	return testMatrixComplete
}
func writeTestMatrixFile(testType string, testMatrix []matrixRow) {
	bytes, err := json.MarshalIndent(testMatrix, "", " ")
	if err != nil {
		log.Panicf("Can't marshal json for target os %v, err %v", testType, err)
	}
	err = ioutil.WriteFile(fmt.Sprintf("generator/resources/%v_complete_test_matrix.json", testType), bytes, os.ModePerm)
	if err != nil {
		log.Panicf("Can't write json to file for target os %v, err %v", testType, err)
	}
}
