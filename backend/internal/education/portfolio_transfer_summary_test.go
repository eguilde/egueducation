package education

import "testing"

func TestPortfolioReadyForTransfer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		readyForReview bool
		documents      int
		transfers      int
		want           bool
	}{
		{name: "complete portfolio with transfer", readyForReview: true, documents: 5, transfers: 1, want: true},
		{name: "review blockers", documents: 5, transfers: 1},
		{name: "no documents", readyForReview: true, transfers: 1},
		{name: "no transfer", readyForReview: true, documents: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completeness := PortfolioCompletenessSummary{ReadyForReview: tt.readyForReview, TotalDocuments: tt.documents}
			transfer := PortfolioTransferSummaryTransfer{TotalEvents: tt.transfers}
			if got := portfolioReadyForTransfer(completeness, transfer); got != tt.want {
				t.Fatalf("portfolioReadyForTransfer() = %v, want %v", got, tt.want)
			}
		})
	}
}
