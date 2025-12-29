package main

import (
	"testing"
)

func TestParseYearMonth(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedYear  int
		expectedMonth int
		expectError   bool
	}{
		{
			name:          "valid year only",
			input:         "2023",
			expectedYear:  2023,
			expectedMonth: 0,
			expectError:   false,
		},
		{
			name:          "valid year and month",
			input:         "06/2023",
			expectedYear:  2023,
			expectedMonth: 6,
			expectError:   false,
		},
		{
			name:          "valid month 12",
			input:         "12/2023",
			expectedYear:  2023,
			expectedMonth: 12,
			expectError:   false,
		},
		{
			name:          "valid month 1",
			input:         "01/2024",
			expectedYear:  2024,
			expectedMonth: 1,
			expectError:   false,
		},
		{
			name:        "invalid year too short",
			input:       "23",
			expectError: true,
		},
		{
			name:        "invalid year too long",
			input:       "20234",
			expectError: true,
		},
		{
			name:        "invalid month 0",
			input:       "00/2023",
			expectError: true,
		},
		{
			name:        "invalid month 13",
			input:       "13/2023",
			expectError: true,
		},
		{
			name:        "invalid month negative",
			input:       "-1/2023",
			expectError: true,
		},
		{
			name:        "invalid format too many parts",
			input:       "06/15/2023",
			expectError: true,
		},
		{
			name:        "invalid format non-numeric year",
			input:       "abc",
			expectError: true,
		},
		{
			name:        "invalid format non-numeric month",
			input:       "abc/2023",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:          "year at lower bound",
			input:         "1000",
			expectedYear:  1000,
			expectedMonth: 0,
			expectError:   false,
		},
		{
			name:          "year at upper bound",
			input:         "9999",
			expectedYear:  9999,
			expectedMonth: 0,
			expectError:   false,
		},
		{
			name:        "year below lower bound",
			input:       "999",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year, month, err := parseYearMonth(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input %q, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error for input %q, got: %v", tt.input, err)
				return
			}

			if year != tt.expectedYear {
				t.Errorf("Expected year %d, got %d", tt.expectedYear, year)
			}

			if month != tt.expectedMonth {
				t.Errorf("Expected month %d, got %d", tt.expectedMonth, month)
			}
		})
	}
}
