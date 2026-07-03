// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package harness

type SenderReport struct {
	TotalMessages  int64
	ThroughputMPS  float64
	MeanLatencyMS  float64
	P50LatencyMS   float64
	P90LatencyMS   float64
	P99LatencyMS   float64
	MaxLatencyMS   float64
	RuntimeErrors  int64
	ActualDuration string
}

type SinkStats struct {
	Mode                 string
	ReceivedMessages     int64
	ReceivedBytes        int64
	ReplyMessages        int64
	Errors               int64
	WarmupMessages       int64
	WarmupReplies        int64
	DrainMessages        int64
	DrainReplies         int64
	ElapsedSeconds       float64
	ActiveReceiveSeconds float64
	ReceiveMPS           float64
	ReceiveMBPS          float64
	ActiveReceiveMPS     float64
	ActiveReceiveMBPS    float64
}
