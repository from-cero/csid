package csid

import (
	"testing"
	"time"
)

func TestApplyOptions_Defaults(t *testing.T) {
	cfg := applyOptions(nil)

	if cfg.Format != DefaultFormat {
		t.Errorf("Format = %+v, want %+v", cfg.Format, DefaultFormat)
	}
	wantEpoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !cfg.Epoch.Equal(wantEpoch) {
		t.Errorf("Epoch = %v, want %v", cfg.Epoch, wantEpoch)
	}
	if cfg.MaxClockDrift != 10*time.Millisecond {
		t.Errorf("MaxClockDrift = %v, want 10ms", cfg.MaxClockDrift)
	}
	if cfg.YieldOnExhaustion {
		t.Error("YieldOnExhaustion = true, want false")
	}
}

func TestWithFormat(t *testing.T) {
	f := Format{TimestampBits: 40, NodeBits: 13, SequenceBits: 10}
	cfg := applyOptions([]Option{WithFormat(f)})
	if cfg.Format != f {
		t.Errorf("Format = %+v, want %+v", cfg.Format, f)
	}
}

func TestWithEpoch(t *testing.T) {
	e := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	cfg := applyOptions([]Option{WithEpoch(e)})
	if !cfg.Epoch.Equal(e) {
		t.Errorf("Epoch = %v, want %v", cfg.Epoch, e)
	}
}

func TestWithMaxClockDrift(t *testing.T) {
	cfg := applyOptions([]Option{WithMaxClockDrift(50 * time.Millisecond)})
	if cfg.MaxClockDrift != 50*time.Millisecond {
		t.Errorf("MaxClockDrift = %v, want 50ms", cfg.MaxClockDrift)
	}
}

func TestWithYieldOnExhaustion(t *testing.T) {
	cfg := applyOptions([]Option{WithYieldOnExhaustion(true)})
	if !cfg.YieldOnExhaustion {
		t.Error("YieldOnExhaustion = false, want true")
	}
}

func TestApplyOptions_MultipleOptions(t *testing.T) {
	e := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := applyOptions(
		[]Option{
			WithEpoch(e),
			WithMaxClockDrift(5 * time.Millisecond),
			WithYieldOnExhaustion(true),
		},
	)
	if !cfg.Epoch.Equal(e) {
		t.Errorf("Epoch = %v, want %v", cfg.Epoch, e)
	}
	if cfg.MaxClockDrift != 5*time.Millisecond {
		t.Errorf("MaxClockDrift = %v, want 5ms", cfg.MaxClockDrift)
	}
	if !cfg.YieldOnExhaustion {
		t.Error("YieldOnExhaustion = false, want true")
	}
}
