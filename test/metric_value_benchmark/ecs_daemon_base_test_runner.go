// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

//go:build !windows

package metric_value_benchmark

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/amazon-cloudwatch-agent-test/test/test_runner"

	"github.com/aws/amazon-cloudwatch-agent-test/environment"
	"github.com/aws/amazon-cloudwatch-agent-test/internal/awsservice"
	"github.com/aws/amazon-cloudwatch-agent-test/test/metric/dimension"
	"github.com/aws/amazon-cloudwatch-agent-test/test/status"
)

type IECSTestRunner interface {
	validate() status.TestGroupResult
	getTestName() string
	getAgentConfigFileName() string
	getAgentRunDuration() time.Duration
	getMeasuredMetrics() []string
}

type IAgentRunStrategy interface {
	runAgent(e *environment.MetaData, configFilePath string) error
}

type ECSAgentRunStrategy struct {
}

func (r *ECSAgentRunStrategy) runAgent(e *environment.MetaData, configFilePath string) error {
	b, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed while reading config file")
	}

	agentConfig := string(b)

	err = awsservice.AWS.SsmAPI.PutStringParameter(e.CwagentConfigSsmParamName, agentConfig)
	if err != nil {
		return fmt.Errorf("failed while reading config file : %s", err.Error())
	}
	fmt.Print("Put parameter successful")

	err = awsservice.AWS.EcsAPI.RestartDaemonService(e.EcsClusterArn, e.EcsServiceName)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print("CWAgent service is restarted")

	time.Sleep(5 * time.Minute)

	return nil
}

type ECSTestRunner struct {
	testRunner       IECSTestRunner
	agentRunStrategy IAgentRunStrategy
	env              environment.MetaData
}

type BaseTestRunner struct {
	DimensionFactory dimension.Factory
}

func (t *ECSTestRunner) Run(s test_runner.ITestSuite, e *environment.MetaData) {
	testName := t.testRunner.getTestName()
	fmt.Printf("Running %s", testName)
	testGroupResult, err := t.runAgent(e)
	if err == nil {
		testGroupResult = t.testRunner.validate()
	}

	s.AddToSuiteResult(testGroupResult)
	if testGroupResult.GetStatus() != status.SUCCESSFUL {
		fmt.Printf("%s test group failed", testName)
	}
}

func (t *ECSTestRunner) runAgent(e *environment.MetaData) (status.TestGroupResult, error) {
	testGroupResult := status.TestGroupResult{
		Name: t.testRunner.getTestName(),
		TestResults: []status.TestResult{
			{
				Name:   "Starting Agent",
				Status: status.FAILED,
			},
		},
	}

	err := t.agentRunStrategy.runAgent(e, t.testRunner.getAgentConfigFileName())

	if err != nil {
		fmt.Print(err)
		return testGroupResult, fmt.Errorf("Failed to run agent with config for the given test")
	}

	testGroupResult.TestResults[0].Status = status.SUCCESSFUL
	return testGroupResult, nil
}
