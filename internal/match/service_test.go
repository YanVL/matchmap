package match

import (
	"math"
	"testing"
)

func TestIsConcordant(t *testing.T) {
	tests := []struct {
		name     string
		result1  string
		result2  string
		expected bool
	}{
		{
			name:     "win and loss",
			result1:  "win",
			result2:  "loss",
			expected: true,
		},
		{
			name:     "loss and win",
			result1:  "loss",
			result2:  "win",
			expected: true,
		},
		{
			name:     "draw and draw",
			result1:  "draw",
			result2:  "draw",
			expected: true,
		},
		{
			name:     "win and win",
			result1:  "win",
			result2:  "win",
			expected: false,
		},
		{
			name:     "loss and loss",
			result1:  "loss",
			result2:  "loss",
			expected: false,
		},
		{
			name:     "win and draw",
			result1:  "win",
			result2:  "draw",
			expected: false,
		},
		{
			name:     "draw and loss",
			result1:  "draw",
			result2:  "loss",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isConcordant(test.result1, test.result2)

			if result != test.expected {
				t.Errorf(
					"expected %v, got %v",
					test.expected,
					result,
				)
			}
		})
	}
}

func TestCalculateMatchScore(t *testing.T) {
	tests := []struct {
		name        string
		result      string
		concordance string
		expected    int
	}{
		{
			name:        "win and concordant",
			result:      "win",
			concordance: "concordant",
			expected:    4,
		},
		{
			name:        "win and discordant",
			result:      "win",
			concordance: "discordant",
			expected:    2,
		},
		{
			name:        "draw and concordant",
			result:      "draw",
			concordance: "concordant",
			expected:    2,
		},
		{
			name:        "draw and discordant",
			result:      "draw",
			concordance: "discordant",
			expected:    0,
		},
		{
			name:        "loss and concordant",
			result:      "loss",
			concordance: "concordant",
			expected:    1,
		},
		{
			name:        "loss and discordant",
			result:      "loss",
			concordance: "discordant",
			expected:    -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := calculateMatchScore(
				test.result,
				test.concordance,
			)

			if result != test.expected {
				t.Errorf(
					"expected %d, got %d",
					test.expected,
					result,
				)
			}
		})
	}
}

func TestCalculateMatchStats(t *testing.T) {
	const tolerance = 0.000001

	tests := []struct {
		name                       string
		results                    []UserMatchResult
		expectedMatches            int
		expectedWins               int
		expectedLosses             int
		expectedDraws              int
		expectedConcordant         int
		expectedDiscordant         int
		expectedScore              int
		expectedWinrate            float64
		expectedConcordanceRate    float64
		expectedWinConcordanceRate float64
	}{
		{
			name: "all concordant",
			results: []UserMatchResult{
				{
					UserResult:     "win",
					OpponentResult: "loss",
				},
				{
					UserResult:     "loss",
					OpponentResult: "win",
				},
				{
					UserResult:     "draw",
					OpponentResult: "draw",
				},
			},
			expectedMatches:            3,
			expectedWins:               1,
			expectedLosses:             1,
			expectedDraws:              1,
			expectedConcordant:         3,
			expectedDiscordant:         0,
			expectedScore:              7,
			expectedWinrate:            33.33333333333333,
			expectedConcordanceRate:    100.0,
			expectedWinConcordanceRate: 100.0,
		},
		{
			name: "one discordant match",
			results: []UserMatchResult{
				{
					UserResult:     "win",
					OpponentResult: "loss",
				},
				{
					UserResult:     "loss",
					OpponentResult: "win",
				},
				{
					UserResult:     "draw",
					OpponentResult: "win",
				},
			},
			expectedMatches:            3,
			expectedWins:               1,
			expectedLosses:             1,
			expectedDraws:              1,
			expectedConcordant:         2,
			expectedDiscordant:         1,
			expectedScore:              5,
			expectedWinrate:            33.33333333333333,
			expectedConcordanceRate:    66.66666666666667,
			expectedWinConcordanceRate: 100.0,
		},
		{
			name: "all discordant",
			results: []UserMatchResult{
				{
					UserResult:     "win",
					OpponentResult: "win",
				},
				{
					UserResult:     "loss",
					OpponentResult: "loss",
				},
				{
					UserResult:     "draw",
					OpponentResult: "win",
				},
			},
			expectedMatches:            3,
			expectedWins:               1,
			expectedLosses:             1,
			expectedDraws:              1,
			expectedConcordant:         0,
			expectedDiscordant:         3,
			expectedScore:              1,
			expectedWinrate:            33.33333333333333,
			expectedConcordanceRate:    0.0,
			expectedWinConcordanceRate: 0.0,
		},
		{
			name:                       "no matches",
			results:                    []UserMatchResult{},
			expectedMatches:            0,
			expectedWins:               0,
			expectedLosses:             0,
			expectedDraws:              0,
			expectedConcordant:         0,
			expectedDiscordant:         0,
			expectedScore:              0,
			expectedWinrate:            0.0,
			expectedConcordanceRate:    0.0,
			expectedWinConcordanceRate: 0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats := calculateMatchStats(test.results)

			if stats.Matches != test.expectedMatches {
				t.Errorf(
					"expected %d matches, got %d",
					test.expectedMatches,
					stats.Matches,
				)
			}

			if stats.Wins != test.expectedWins {
				t.Errorf(
					"expected %d wins, got %d",
					test.expectedWins,
					stats.Wins,
				)
			}

			if stats.Losses != test.expectedLosses {
				t.Errorf(
					"expected %d losses, got %d",
					test.expectedLosses,
					stats.Losses,
				)
			}

			if stats.Draws != test.expectedDraws {
				t.Errorf(
					"expected %d draws, got %d",
					test.expectedDraws,
					stats.Draws,
				)
			}

			if stats.ConcordantMatches != test.expectedConcordant {
				t.Errorf(
					"expected %d concordant matches, got %d",
					test.expectedConcordant,
					stats.ConcordantMatches,
				)
			}

			if stats.DiscordantMatches != test.expectedDiscordant {
				t.Errorf(
					"expected %d discordant matches, got %d",
					test.expectedDiscordant,
					stats.DiscordantMatches,
				)
			}

			if stats.Score != test.expectedScore {
				t.Errorf(
					"expected score %d, got %d",
					test.expectedScore,
					stats.Score,
				)
			}

			if math.Abs(stats.Winrate-test.expectedWinrate) > tolerance {
				t.Errorf(
					"expected winrate %f, got %f",
					test.expectedWinrate,
					stats.Winrate,
				)
			}

			if math.Abs(stats.ConcordanceRate-test.expectedConcordanceRate) > tolerance {
				t.Errorf(
					"expected concordance rate %f, got %f",
					test.expectedConcordanceRate,
					stats.ConcordanceRate,
				)
			}

			if math.Abs(stats.WinConcordanceRate-test.expectedWinConcordanceRate) > tolerance {
				t.Errorf(
					"expected win concordance rate %f, got %f",
					test.expectedWinConcordanceRate,
					stats.WinConcordanceRate,
				)
			}
		})
	}
}
