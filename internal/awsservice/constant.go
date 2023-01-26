// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package awsservice

import (
	"time"

	backoff "github.com/cenkalti/backoff/v4"
)

const (
	StandardRetries = 5
)

var (
	StandardExponentialBackoff = backoff.WithMaxRetries(&backoff.ExponentialBackOff{
		InitialInterval:     30 * time.Second,
		RandomizationFactor: 2,
		Multiplier:          2,
		MaxInterval:         60 * time.Second,
		MaxElapsedTime:      30 * time.Second,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}, StandardRetries)
)
