package regulatorysource

import "testing"

func TestNewServiceUnconfiguredFetcher(t *testing.T) {
	t.Parallel()
	var absent *Fetcher
	tests := []struct {
		name    string
		fetcher evidenceFetcher
	}{
		{name: "nil interface"},
		{name: "nil concrete pointer", fetcher: absent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if NewService(nil, tt.fetcher).fetcher != nil {
				t.Fatal("unconfigured publisher fetcher must remain unavailable")
			}
		})
	}
}
