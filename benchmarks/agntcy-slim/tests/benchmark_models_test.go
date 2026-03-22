package tests

import "time"

type suiteConfig struct {
	OutputDir                     string
	RawDir                        string
	TemplateDir                   string
	SummaryFile                   string
	TechnicalReportFile           string
	ResultsTSV                    string
	CapacitySweepFile             string
	Sizes                         []int
	Clients                       []int
	Modes                         []string
	RequestRates                  []int
	PubRates                      []int
	WriteRates                    []int
	PubRatesDisplay               string
	WriteRatesDisplay             string
	PubRateAutoProfile            bool
	Duration                      time.Duration
	DurationDisplay               string
	Repeats                       int
	ServerEndpoint                string
	Destination                   string
	ModesDisplay                  string
	ClientsDisplay                string
	SizesDisplay                  string
	RequestRatesDisplay           string
	CapacitySweepEnabled          bool
	CapacitySweepModes            []string
	CapacitySweepClients          []int
	CapacitySweepSizes            []int
	CapacitySweepStartRate        int
	CapacitySweepMaxRate          int
	CapacitySweepGrowthFactor     float64
	CapacitySweepPlateauThreshold float64
	CapacitySweepPlateauSteps     int
	CapacitySweepMaxSteps         int
	CapacitySweepRepeats          int
	CapacitySweepRefinementSteps  int
	CapacitySweepMinRateDelta     int
	CapacitySweepModesDisplay     string
	CapacitySweepClientsDisplay   string
	CapacitySweepSizesDisplay     string
}

type benchmarkRunResult struct {
	Mode                     string
	Clients                  int
	Size                     int
	Rate                     int
	Repeat                   int
	SenderTotalMessages      int64
	SenderMPS                float64
	SenderMeanLatencyMS      float64
	SenderP50LatencyMS       float64
	SenderP90LatencyMS       float64
	SenderP99LatencyMS       float64
	SenderMaxLatencyMS       float64
	SenderRuntimeErrors      int64
	SenderDuration           string
	SinkReceivedMessages     int64
	SinkErrors               int64
	SinkReceiveMPS           float64
	SinkActiveReceiveMPS     float64
	SinkElapsedSeconds       float64
	SinkActiveReceiveSeconds float64
	SenderCPUSeconds         float64
	SenderCPUPercent         float64
	ResponderCPUSeconds      float64
	ResponderCPUPercent      float64
	NodeCPUSeconds           float64
	NodeCPUPercent           float64
	TotalCPUSeconds          float64
	TotalCPUPercent          float64
}

type processCPUUsage struct {
	SenderCPUSeconds    float64
	SenderCPUPercent    float64
	ResponderCPUSeconds float64
	ResponderCPUPercent float64
	NodeCPUSeconds      float64
	NodeCPUPercent      float64
	TotalCPUSeconds     float64
	TotalCPUPercent     float64
}

type senderReport struct {
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

type sinkStats struct {
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

type capacitySweepStepResult struct {
	Phase               string
	Step                int
	Rate                int
	Repeats             int
	SenderMeanMPS       float64
	SenderVariance      float64
	SenderCILow         float64
	SenderCIHigh        float64
	ObservedMeanMPS     float64
	ObservedVariance    float64
	ObservedCILow       float64
	ObservedCIHigh      float64
	NodeMeanCPUPercent  float64
	NodeVariance        float64
	NodeCILow           float64
	NodeCIHigh          float64
	TotalMeanCPUPercent float64
	TotalVariance       float64
	TotalCILow          float64
	TotalCIHigh         float64
	TotalErrors         int64
	ObservedGainPercent float64
	Improved            bool
}

type capacitySweepCaseResult struct {
	Mode                string
	Clients             int
	Size                int
	BestRate            int
	CapacityRateLower   int
	CapacityRateUpper   int
	BestObservedMeanMPS float64
	BestObservedCILow   float64
	BestObservedCIHigh  float64
	BestSenderMeanMPS   float64
	BestSenderCILow     float64
	BestSenderCIHigh    float64
	BestNodeCPUPercent  float64
	BestNodeCILow       float64
	BestNodeCIHigh      float64
	BestTotalCPUPercent float64
	BestTotalCILow      float64
	BestTotalCIHigh     float64
	Steps               []capacitySweepStepResult
	StopReason          string
}
