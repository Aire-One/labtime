package monitors

import "testing"

func TestHTTPMonitor_ID(t *testing.T) {
	monitor := &HTTPMonitor{
		Label: "example",
	}

	expectedID := "example"
	actualID := monitor.ID()

	if actualID != expectedID {
		t.Errorf("expected ID to be %s, but got %s", expectedID, actualID)
	}
}
