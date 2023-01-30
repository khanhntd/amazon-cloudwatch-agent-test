// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package stress

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/amazon-cloudwatch-agent-test/internal/awsservice"
	"github.com/aws/amazon-cloudwatch-agent-test/internal/common"
	"github.com/aws/amazon-cloudwatch-agent-test/validator/models"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go/aws"
	"go.uber.org/multierr"
)

// Todo:
// * Create a database  to store these metrics instead of using cache?
// * Create a workflow to update the bound metrics?
// * Increase data rate to even a high loader (e.g 1000, 5000, 10000)
var (
	metricErrorBound = 0.3
	metricBoundValue = map[string]map[string]float64{
		"10": {
			"procstat_cpu_usage":   float64(5),
			"procstat_memory_rss":  float64(66500000),
			"procstat_memory_swap": float64(0),
			"procstat_memory_vms":  float64(818000000),
			"procstat_num_fds":     float64(9),
			"procstat_write_bytes": float64(150000),
			"net_bytes_sent":       float64(14000),
			"net_packets_sent":     float64(35),
		},
	}
)

type StressValidator struct {
	vConfig models.ValidateConfig
}

var _ models.ValidatorFactory = (*StressValidator)(nil)

func NewStressValidator(vConfig models.ValidateConfig) models.ValidatorFactory {
	return &StressValidator{
		vConfig: vConfig,
	}
}

func (s *StressValidator) InitValidation() (err error) {
	var (
		datapointPeriod     = s.vConfig.GetDataPointPeriod()
		agentConfigFilePath = s.vConfig.GetCloudWatchAgentConfigPath()
		dataType            = s.vConfig.GetDataType()
		dataRate            = s.vConfig.GetDataRate()
		receivers, _, _     = s.vConfig.GetOtelConfig()
	)
	switch dataType {
	case "logs":
		err = common.StartLogWrite(agentConfigFilePath, datapointPeriod, dataRate)
	default:
		err = common.StartSendingMetrics(receivers, datapointPeriod, dataRate)
	}

	return err
}

func (s *StressValidator) StartValidation(startTime, endTime time.Time) error {
	var (
		multiErr         error
		ec2InstanceId    = awsservice.GetInstanceId()
		metricNamespace  = s.vConfig.GetMetricNamespace()
		validationMetric = s.vConfig.GetMetricValidation()
	)
	for _, metric := range validationMetric {
		metricDimensions := []types.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(ec2InstanceId),
			},
		}
		for _, dimension := range metric.MetricDimension {
			metricDimensions = append(metricDimensions, types.Dimension{
				Name:  aws.String(dimension.Name),
				Value: aws.String(dimension.Value),
			})
		}
		err := s.ValidateStressMetric(metric.MetricName, metricNamespace, metricDimensions, startTime, endTime)
		if err != nil {
			multiErr = multierr.Append(multiErr, err)
		}
	}

	return multiErr
}

func (s *StressValidator) EndValidation() error {
	var (
		dataType      = s.vConfig.GetDataType()
		ec2InstanceId = awsservice.GetInstanceId()
	)
	switch dataType {
	case "logs":
		awsservice.DeleteLogGroup(ec2InstanceId)
	}

	return nil
}

func (s *StressValidator) ValidateStressMetric(metricName, metricNamespace string, metricDimensions []types.Dimension, startTime, endTime time.Time) error {
	var (
		dataRate       = fmt.Sprint(s.vConfig.GetDataRate())
		boundAndPeriod = s.vConfig.GetDataPointPeriod().Seconds()
	)

	stressMetricQueries := s.buildStressMetricQueries(metricName, metricNamespace, metricDimensions)

	log.Printf("Start to collect and validate metric %s with the namespace %s, start time %v and end time %v", metricName, metricNamespace, startTime, endTime)

	// We are only interesting in the maxium metric values within the time range since the metrics sending are not distributed evenly; therefore,
	// only the maximum shows the correct usage reflection of CloudWatchAgent during that time
	metrics, err := awsservice.GetMetricData(stressMetricQueries, startTime, endTime)
	if err != nil {
		return err
	}

	if len(metrics.MetricDataResults) == 0 || len(metrics.MetricDataResults[0].Values) == 0 {
		return fmt.Errorf("getting metric %s failed with the namespace %s and dimension %v", metricName, metricNamespace, metricDimensions)
	}

	if _, ok := metricBoundValue[dataRate]; !ok {
		return fmt.Errorf("metric %s does not have data rate", metricName)
	}

	if _, ok := metricBoundValue[dataRate][metricName]; !ok {
		return fmt.Errorf("metric %s does not have bound", metricName)
	}

	// Validate if the corresponding metrics are within the acceptable range [acceptable value +- 30%]
	metricValue := metrics.MetricDataResults[0].Values[0]
	lowerBoundValue := metricBoundValue[dataRate][metricName] * (1 - metricErrorBound)
	upperBoundValue := metricBoundValue[dataRate][metricName] * (1 + metricErrorBound)
	if metricValue < 0 || metricValue > upperBoundValue || metricValue < lowerBoundValue {
		return fmt.Errorf("metric %s with value %f is not within bound [ %f, %f ] ", metricName, metricValue, lowerBoundValue, upperBoundValue)
	}

	// Validate if the metrics are not dropping any metrics and able to backfill within the same minute (e.g if the memory_rss metric is having collection_interval 1
	// , it will need to have 60 sample counts - 1 datapoint / second)
	if ok := awsservice.ValidateSampleCount(metricName, metricNamespace, metricDimensions, startTime, endTime, int(boundAndPeriod-1), int(boundAndPeriod+1), int32(boundAndPeriod)); !ok {
		return fmt.Errorf("metric %s is not within sample count bound [ %f, %f]", metricName, boundAndPeriod, boundAndPeriod)
	}

	return nil
}

func (s *StressValidator) buildStressMetricQueries(metricName, metricNamespace string, metricDimensions []types.Dimension) []types.MetricDataQuery {
	var (
		metricQueryPeriod = int32(s.vConfig.GetDataPointPeriod().Seconds())
	)

	metricInformation := types.Metric{
		Namespace:  aws.String(metricNamespace),
		MetricName: aws.String(metricName),
		Dimensions: metricDimensions,
	}

	metricDataQueries := []types.MetricDataQuery{
		{
			MetricStat: &types.MetricStat{
				Metric: &metricInformation,
				Period: &metricQueryPeriod,
				Stat:   aws.String(string(models.MAXIMUM)),
			},
			Id: aws.String(strings.ToLower(metricName)),
		},
	}
	return metricDataQueries
}
