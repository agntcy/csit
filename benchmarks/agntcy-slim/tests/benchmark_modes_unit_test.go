package tests

import (
	"reflect"
	"testing"

	"github.com/onsi/gomega"
)

func TestCanonicalBenchmarkMode(t *testing.T) {
	gomega.RegisterTestingT(t)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "request reply", input: "request-reply", want: "request-reply"},
		{name: "fire and forget uppercase", input: " FIRE-AND-FORGET ", want: "fire-and-forget"},
		{name: "write", input: "write", want: "write"},
		{name: "unsupported", input: "pub", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := canonicalBenchmarkMode(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestModeRateValues(t *testing.T) {
	gomega.RegisterTestingT(t)

	cfg := suiteConfig{
		RequestRates: []int{100},
		PubRates:     []int{200},
		WriteRates:   []int{300},
	}

	if got := modeRateValues(cfg, "request-reply", 1, 16); !reflect.DeepEqual(got, []int{100}) {
		t.Fatalf("request-reply rates = %v, want %v", got, []int{100})
	}
	if got := modeRateValues(cfg, "write", 1, 16); !reflect.DeepEqual(got, []int{300}) {
		t.Fatalf("write rates = %v, want %v", got, []int{300})
	}

	cfg.WriteRates = nil
	if got := modeRateValues(cfg, "write", 1, 16); !reflect.DeepEqual(got, []int{200}) {
		t.Fatalf("write fallback rates = %v, want %v", got, []int{200})
	}
}

func TestBenchmarkModeMetadata(t *testing.T) {
	gomega.RegisterTestingT(t)

	if !modeUsesSinkMetrics("request-reply") {
		t.Fatal("request-reply should use sink metrics")
	}
	if !modeUsesSinkMetrics("fire-and-forget") {
		t.Fatal("fire-and-forget should use sink metrics")
	}
	if modeUsesSinkMetrics("write") {
		t.Fatal("write should not use sink metrics")
	}
	if got := modeResponderKind("write"); got != "blackhole" {
		t.Fatalf("write responder = %q, want blackhole", got)
	}
	if got := modeObservedThroughputLabel("write"); got != "Sender Write Throughput" {
		t.Fatalf("write observed label = %q", got)
	}
}
