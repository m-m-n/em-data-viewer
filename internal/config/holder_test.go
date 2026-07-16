package config

import (
	"errors"
	"reflect"
	"testing"
)

// TestHolder_RetainsLastGoodOnFailedReload references AC-5.
func TestHolder_RetainsLastGoodOnFailedReload(t *testing.T) {
	h := NewHolder()

	good := Config{Window: Window{Width: 400, Height: 600}}
	h.Update(good, nil)

	if got := h.Config(); got.Window.Width != 400 || got.Window.Height != 600 {
		t.Fatalf("Config() = %+v, want initial good config", got)
	}
	if err := h.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil after successful load", err)
	}

	failErr := errors.New("boom")
	h.Update(Config{}, failErr)

	if got := h.Config(); got.Window.Width != 400 || got.Window.Height != 600 {
		t.Errorf("Config() after failed reload = %+v, want previous good config retained", got)
	}
	if err := h.Err(); err != failErr {
		t.Errorf("Err() after failed reload = %v, want %v", err, failErr)
	}
}

// TestHolder_SuccessfulReloadReplacesConfigAndClearsError references AC-5.
func TestHolder_SuccessfulReloadReplacesConfigAndClearsError(t *testing.T) {
	h := NewHolder()
	h.Update(Config{}, errors.New("initial failure"))
	if h.Err() == nil {
		t.Fatal("Err() = nil, want initial failure recorded")
	}

	newGood := Config{Window: Window{Width: 999, Height: 111}}
	h.Update(newGood, nil)

	if got := h.Config(); got.Window.Width != 999 || got.Window.Height != 111 {
		t.Errorf("Config() = %+v, want new good config %+v", got, newGood)
	}
	if err := h.Err(); err != nil {
		t.Errorf("Err() = %v, want nil after successful reload", err)
	}
}

func TestHolder_ZeroValueHasNoConfigOrErrorBeforeAnyUpdate(t *testing.T) {
	h := NewHolder()
	if err := h.Err(); err != nil {
		t.Errorf("Err() = %v, want nil before any Update", err)
	}
	if got := h.Config(); !reflect.DeepEqual(got, Config{}) {
		t.Errorf("Config() = %+v, want zero value before any Update", got)
	}
}
