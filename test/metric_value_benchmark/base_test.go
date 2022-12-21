// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

package metric_value_benchmark

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/aws/amazon-cloudwatch-agent-test/internal/common"
	"github.com/aws/amazon-cloudwatch-agent-test/test/metric"
	"github.com/aws/amazon-cloudwatch-agent-test/test/status"
)

const (
	configOutputPath     = "/opt/aws/amazon-cloudwatch-agent/bin/config.json"
	agentConfigDirectory = "agent_configs"
	extraConfigDirectory = "extra_configs"
	minimumAgentRuntime  = 3 * time.Minute
)

type ITestRunner interface {
	validate() status.TestGroupResult
	getTestName() string
	getAgentConfigFileName() string
	getAgentRunDuration() time.Duration
	getMeasuredMetrics() []string
	setupAfterAgentRun() error
}

type TestRunner struct {
	testRunner ITestRunner
}

type BaseTestRunner struct {
	*metric.MetricFetcherFactory
}

func (t *TestRunner) Run(s *MetricBenchmarkTestSuite) {
	testName := t.testRunner.getTestName()
	log.Printf("Running %v", testName)
	testGroupResult, err := t.runAgent()
	if err == nil {
		testGroupResult = t.testRunner.validate()
	}
	s.AddToSuiteResult(testGroupResult)
	if testGroupResult.GetStatus() != status.SUCCESSFUL {
		log.Printf("%v test group failed", testName)
	}
}

func (t *TestRunner) runAgent() (status.TestGroupResult, error) {
	testGroupResult := status.TestGroupResult{
		Name: t.testRunner.getTestName(),
		TestResults: []status.TestResult{
			{
				Name:   "Starting Agent",
				Status: status.SUCCESSFUL,
			},
		},
	}

	agentConfigPath := filepath.Join(agentConfigDirectory, t.testRunner.getAgentConfigFileName())
	log.Printf("Starting agent using agent config file %s", agentConfigPath)
	common.CopyFile(agentConfigPath, configOutputPath)
	err := common.StartAgent(configOutputPath, false)

	if err != nil {
		testGroupResult.TestResults[0].Status = status.FAILED
		return testGroupResult, fmt.Errorf("Agent could not start due to: %s", err.Error())
	}

	err = t.testRunner.setupAfterAgentRun()
	if err != nil {
		testGroupResult.TestResults[0].Status = status.FAILED
		return testGroupResult, fmt.Errorf("Failed to run extra commands due to: %s", err.Error())
	}

	runningDuration := t.testRunner.getAgentRunDuration()
	time.Sleep(runningDuration)
	log.Printf("Agent has been running for : %s", runningDuration.String())
	common.StopAgent()

	err = common.DeleteFile(configOutputPath)
	if err != nil {
		testGroupResult.TestResults[0].Status = status.FAILED
		return testGroupResult, fmt.Errorf("Failed to cleanup config file after agent run due to: %s", err.Error())
	}

	return testGroupResult, nil
}
