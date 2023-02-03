// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package models // import "github.com/aws/amazon-cloudwatch-agent-test/validator/models"

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type ValidateConfig interface {
	GetOtelConfig() ([]string, []string, []string)
	GetValidateType() string
	GetTestCase() string
	GetDataType() string
	GetDataRate() int
	GetCloudWatchAgentConfigPath() string
	GetDataPointPeriod() time.Duration
	GetMetricNamespace() string
	GetMetricValidation() []MetricValidation
	GetCommitInformation() (string, int64)
}
type validatorConfig struct {
	Receivers  []string `yaml:"receivers"`
	Processors []string `yaml:"processors"`
	Exporters  []string `yaml:"exporters"`

	TestCase        string `yaml:"test_case"`
	ValidateType    string `yaml:"validate_type"`
	DataType        string `yaml:"data_type"`
	DataRate        string `yaml:"data_rate"`
	DatapointPeriod int    `yaml:"datapoint_period"`

	ConfigPath string `yaml:"cloudwatch_agent_config"`

	MetricNamespace  string             `yaml:"metric_namespace"`
	MetricValidation []MetricValidation `yaml:"metric_validation"`

	CommitHash string `yaml:"commit_hash"`
	CommitDate string `yaml:"commit_date"`
}

type MetricValidation struct {
	MetricName      string            `yaml:"metric_name"`
	MetricDimension []MetricDimension `yaml:"metric_dimension"`
}

type MetricDimension struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func NewValidateConfig(configPath string) (*validatorConfig, error) {
	configPathBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("%v with file %s", err, configPath)
	}

	vConfig := validatorConfig{}
	err = yaml.Unmarshal(configPathBytes, &vConfig)
	if err != nil {
		return nil, err
	}
	log.Printf("Parameters validation for %v", vConfig)
	return &vConfig, nil
}

func (v *validatorConfig) GetTestCase() string {
	return v.TestCase
}

func (v *validatorConfig) GetValidateType() string {
	return v.ValidateType
}

func (v *validatorConfig) GetOtelConfig() ([]string, []string, []string) {
	return v.Receivers, v.Processors, v.Exporters
}

func (v *validatorConfig) GetDataType() string {
	return v.DataType
}

func (v *validatorConfig) GetDataRate() int {
	if dataRate, err := strconv.ParseInt(v.DataRate, 10, 64); err == nil {
		return int(dataRate)
	}
	return 0
}

func (v *validatorConfig) GetCloudWatchAgentConfigPath() string {
	return v.ConfigPath
}

func (v *validatorConfig) GetDataPointPeriod() time.Duration {
	return time.Duration(v.DatapointPeriod) * time.Second
}

func (v *validatorConfig) GetMetricNamespace() string {
	return v.MetricNamespace
}

func (v *validatorConfig) GetMetricValidation() []MetricValidation {
	return v.MetricValidation
}

func (v *validatorConfig) GetCommitInformation() (string, int64) {
	commitDate, _ := strconv.ParseInt(v.CommitDate, 10, 64)
	return v.CommitHash, commitDate
}
