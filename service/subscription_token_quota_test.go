package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptionTokenQuotaUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		prompt       int
		completion   int
		cache        int
		includeCache bool
		want         int64
	}{
		{
			name:         "include cache matches model stats p+c+cache",
			prompt:       304544,
			completion:   12781,
			cache:        303616,
			includeCache: true,
			want:         620941,
		},
		{
			name:         "exclude cache is p+c only",
			prompt:       304544,
			completion:   12781,
			cache:        303616,
			includeCache: false,
			want:         317325,
		},
		{
			name:         "zero cache is a no-op when include",
			prompt:       100,
			completion:   20,
			cache:        0,
			includeCache: true,
			want:         120,
		},
		{
			name:         "negative inputs clamp to non-negative total",
			prompt:       0,
			completion:   0,
			cache:        0,
			includeCache: true,
			want:         0,
		},
		{
			name:         "small request with cache",
			prompt:       1000,
			completion:   50,
			cache:        400,
			includeCache: true,
			want:         1450,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := subscriptionTokenQuotaUsage(tt.prompt, tt.completion, tt.cache, tt.includeCache)
			assert.Equal(t, tt.want, got)
		})
	}
}
