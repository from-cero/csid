package csid

import (
	"testing"
	"time"
)

func TestApplyOptions_Defaults(t *testing.T) {
	cfg := applyOptions(nil)

	if cfg.format != defaultFormat {
		t.Errorf("Format = %+v, want %+v", cfg.format, defaultFormat)
	}
	wantEpoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !cfg.epoch.Equal(wantEpoch) {
		t.Errorf("Epoch = %v, want %v", cfg.epoch, wantEpoch)
	}
	if cfg.maxClockDrift != 10*time.Millisecond {
		t.Errorf("MaxClockDrift = %v, want 10ms", cfg.maxClockDrift)
	}
	if cfg.yieldOnExhaustion {
		t.Error("YieldOnExhaustion = true, want false")
	}
}

func TestWithFormat(t *testing.T) {
	cfg := applyOptions([]Option{WithFormat(WithTimestampBits(40), WithNodeBits(13), WithSequenceBits(10))})
	want := format{timestampBits: 40, nodeBits: 13, sequenceBits: 10}
	if cfg.format != want {
		t.Errorf("Format = %+v, want %+v", cfg.format, want)
	}
}

func TestWithEpoch(t *testing.T) {
	e := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	cfg := applyOptions([]Option{WithEpoch(e)})
	if !cfg.epoch.Equal(e) {
		t.Errorf("Epoch = %v, want %v", cfg.epoch, e)
	}
}

func TestWithMaxClockDrift(t *testing.T) {
	cfg := applyOptions([]Option{WithMaxClockDrift(50 * time.Millisecond)})
	if cfg.maxClockDrift != 50*time.Millisecond {
		t.Errorf("MaxClockDrift = %v, want 50ms", cfg.maxClockDrift)
	}
}

func TestWithYieldOnExhaustion(t *testing.T) {
	cfg := applyOptions([]Option{WithYieldOnExhaustion(true)})
	if !cfg.yieldOnExhaustion {
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
	if !cfg.epoch.Equal(e) {
		t.Errorf("Epoch = %v, want %v", cfg.epoch, e)
	}
	if cfg.maxClockDrift != 5*time.Millisecond {
		t.Errorf("MaxClockDrift = %v, want 5ms", cfg.maxClockDrift)
	}
	if !cfg.yieldOnExhaustion {
		t.Error("YieldOnExhaustion = false, want true")
	}
}
