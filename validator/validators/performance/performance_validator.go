// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package performance

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/amazon-cloudwatch-agent-test/internal/awsservice"
	"github.com/aws/amazon-cloudwatch-agent-test/internal/common"
	"github.com/aws/amazon-cloudwatch-agent-test/validator/models"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/cenkalti/backoff/v4"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const (
	ServiceName      = "AmazonCloudWatchAgent"
	DynamoDBDataBase = "CWAPerformanceMetrics"
)

var (
	metricsConvertToMB = []string{"mem_total", "procstat_memory_rss", "procstat_memory_swap", "procstat_memory_data", "procstat_memory_vms", "procstat_write_bytes", "procstat_bytes_sent"}
)

type PerformanceValidator struct {
	vConfig models.ValidateConfig
}

var _ models.ValidatorFactory = (*PerformanceValidator)(nil)

func NewPerformanceValidator(vConfig models.ValidateConfig) models.ValidatorFactory {
	return &PerformanceValidator{
		vConfig: vConfig,
	}
}

func (s *PerformanceValidator) GenerateLoad() (err error) {
	var (
		agentCollectionPeriod = s.vConfig.GetAgentCollectionPeriod()
		agentConfigFilePath   = s.vConfig.GetCloudWatchAgentConfigPath()
		dataType              = s.vConfig.GetDataType()
		dataRate              = s.vConfig.GetDataRate()
		receiver              = s.vConfig.GetPluginsConfig()
	)
	switch dataType {
	case "logs":
		err = common.StartLogWrite(agentConfigFilePath, agentCollectionPeriod, dataRate)
	default:
		err = common.StartSendingMetrics(receiver, agentCollectionPeriod, dataRate)
	}

	return err
}

func (s *PerformanceValidator) CheckData(startTime, endTime time.Time) error {
	metrics, err := s.GetPerformanceMetrics(startTime, endTime)
	if err != nil {
		return err
	}

	perfInfo, err := s.CalculateMetricStatsAndPackMetrics(metrics.MetricDataResults)
	if err != nil {
		return err
	}

	err = s.SendPacketToDatabase(perfInfo)
	if err != nil {
		return err
	}
	return nil
}

func (s *PerformanceValidator) Cleanup() error {
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

func (s *PerformanceValidator) SendPacketToDatabase(perfInfo PerformanceInformation) error {
	var (
		dataType               = s.vConfig.GetDataType()
		receiver               = s.vConfig.GetPluginsConfig()
		commitHash, commitDate = s.vConfig.GetCommitInformation()
		agentCollectionPeriod  = fmt.Sprint(s.vConfig.GetAgentCollectionPeriod().Seconds())
		// The secondary global index that is used for checking if there are item has already been exist in the table
		kCheckingAttribute = []string{"CommitDate", "UseCase"}
		vCheckingAttribute = []string{fmt.Sprint(commitDate), receiver}
	)

	err := backoff.Retry(func() error {
		existingPerfInfo, err := awsservice.GetPacketInDatabase(DynamoDBDataBase, "UseCaseDate", kCheckingAttribute, vCheckingAttribute, perfInfo)
		if err != nil {
			return err
		}

		// Get the latest performance information from the database and update by merging the existing one
		// and finally replace the packet in the database
		maps.Copy(existingPerfInfo["Results"].(map[string]interface{}), perfInfo["Results"].(map[string]interface{}))

		uniqueID := existingPerfInfo["UniqueID"].(string)
		finalPerfInfo := packIntoPerformanceInformation(uniqueID, receiver, dataType, agentCollectionPeriod, commitHash, commitDate, existingPerfInfo["Results"])

		err = awsservice.ReplacePacketInDatabase(DynamoDBDataBase, finalPerfInfo)

		if err != nil {
			return err
		}
		return nil
	}, awsservice.StandardExponentialBackoff)

	return err
}
func (s *PerformanceValidator) CalculateMetricStatsAndPackMetrics(metrics []types.MetricDataResult) (PerformanceInformation, error) {
	var (
		uniqueID               = s.vConfig.GetUniqueID()
		receiver               = s.vConfig.GetPluginsConfig()
		commitHash, commitDate = s.vConfig.GetCommitInformation()
		dataType               = s.vConfig.GetDataType()
		dataRate               = fmt.Sprint(s.vConfig.GetDataRate())
		agentCollectionPeriod  = s.vConfig.GetAgentCollectionPeriod().Seconds()
	)
	performanceMetricResults := make(map[string]Stats)

	for _, metric := range metrics {
		metricLabel := strings.Split(*metric.Label, " ")
		metricName := metricLabel[len(metricLabel)-1]
		metricValues := metric.Values
		//Convert every bytes to MB
		if slices.Contains(metricsConvertToMB, metricName) {
			for i, val := range metricValues {
				metricValues[i] = val / (1000000)
			}
		}
		log.Printf("Start calculate metric statictics for metric %s %v", metricName, metricValues)
		if !isAllValuesGreaterThanOrEqualToZero(metricValues) {
			return nil, fmt.Errorf("values are not all greater than or equal to zero for metric %s with values: %v", metricName, metricValues)
		}
		metricStats := CalculateMetricStatisticsBasedOnDataAndPeriod(metricValues, agentCollectionPeriod)
		log.Printf("Finished calculate metric statictics for metric %s: %v", metricName, metricStats)
		performanceMetricResults[metricName] = metricStats
	}

	return packIntoPerformanceInformation(uniqueID, receiver, dataType, fmt.Sprint(agentCollectionPeriod), commitHash, commitDate, map[string]interface{}{dataRate: performanceMetricResults}), nil
}

func (s *PerformanceValidator) GetPerformanceMetrics(startTime, endTime time.Time) (*cloudwatch.GetMetricDataOutput, error) {
	var (
		metricNamespace              = s.vConfig.GetMetricNamespace()
		validationMetric             = s.vConfig.GetMetricValidation()
		ec2InstanceId                = awsservice.GetInstanceId()
		performanceMetricDataQueries = []types.MetricDataQuery{}
	)
	log.Printf("Start getting performance metrics from CloudWatch")
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
		performanceMetricDataQueries = append(performanceMetricDataQueries, s.buildStressMetricQueries(metric.MetricName, metricNamespace, metricDimensions))
	}

	metrics, err := awsservice.GetMetricData(performanceMetricDataQueries, startTime, endTime)

	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (s *PerformanceValidator) buildStressMetricQueries(metricName, metricNamespace string, metricDimensions []types.Dimension) types.MetricDataQuery {
	metricInformation := types.Metric{
		Namespace:  aws.String(metricNamespace),
		MetricName: aws.String(metricName),
		Dimensions: metricDimensions,
	}

	metricDataQuery := types.MetricDataQuery{
		MetricStat: &types.MetricStat{
			Metric: &metricInformation,
			Period: aws.Int32(10),
			Stat:   aws.String(string(models.AVERAGE)),
		},
		Id: aws.String(strings.ToLower(metricName)),
	}
	return metricDataQuery
}

func packIntoPerformanceInformation(uniqueID, receiver, dataType, collectionPeriod, commitHash string, commitDate int64, result interface{}) PerformanceInformation {
	instanceAMI := awsservice.GetImageId()
	instanceType := awsservice.GetInstanceType()

	return PerformanceInformation{
		"Service":          ServiceName,
		"UniqueID":         uniqueID,
		"UseCase":          receiver,
		"CommitDate":       commitDate,
		"CommitHash":       commitHash,
		"DataType":         dataType,
		"Results":          result,
		"CollectionPeriod": collectionPeriod,
		"InstanceAMI":      instanceAMI,
		"InstanceType":     instanceType,
	}
}

func isAllValuesGreaterThanOrEqualToZero(values []float64) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value < 0 {
			return false
		}
	}
	return true
}
