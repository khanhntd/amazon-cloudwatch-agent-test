// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package awsservice

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

var identityDocument *imds.GetInstanceIdentityDocumentOutput

func GetInstanceId() string {
	return GetImdsMetadata().InstanceID
}

func GetImageId() string {
	return GetImdsMetadata().ImageID
}
func GetInstanceType() string {
	return GetImdsMetadata().InstanceType
}

// TODO: Refactor Structure and Interface for more easier follow that shares the same session
func GetImdsMetadata() *imds.GetInstanceIdentityDocumentOutput {
	identityDocument, err := ImdsClient.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		log.Fatalf("Error occurred while retrieving imds identityDoc: %v", err)
	}
	return identityDocument
}
