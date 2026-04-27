package fund

import "context"

type Provider interface {
	Search(ctx context.Context, query string, limit int) ([]SearchCandidate, error)
	GetProfile(ctx context.Context, code string) (*Profile, error)
	GetTrend(ctx context.Context, code string, rangeKey string) (*Trend, error)
}
