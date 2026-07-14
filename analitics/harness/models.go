// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package harness

type SenderReport struct {
	TotalMessages int64
	ThroughputMPS float64
	MeanLatencyMS float64
	RuntimeErrors int64
}

type SinkStats struct {
	ReceivedMessages int64
	ReplyMessages    int64
	Errors           int64
}
