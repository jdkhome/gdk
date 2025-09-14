package time36

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeToTime36(t *testing.T) {
	testCases := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "base time (2020-01-01 00:00:00)",
			input:    baseTime,
			expected: "00000000",
		},
		{
			name:     "1 millisecond after base time",
			input:    baseTime.Add(1 * time.Millisecond),
			expected: "00000001",
		},
		{
			name:     "35 milliseconds after base time",
			input:    baseTime.Add(35 * time.Millisecond),
			expected: "0000000Z",
		},
		{
			name:     "36 milliseconds after base time",
			input:    baseTime.Add(36 * time.Millisecond),
			expected: "00000010",
		},
		{
			name:     "1 second after base time",
			input:    baseTime.Add(1 * time.Second),
			expected: "000000RS",
		},
		{
			name:     "specific date: 2020-01-02 00:00:00",
			input:    time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC),
			expected: "001FFUO0", // 86400000 ms = 86400000 in base36 is 151800
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := TimeToTime36(tc.input)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTime36ToTime(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected time.Time
		hasError bool
	}{
		{
			name:     "base time (00000000)",
			input:    "00000000",
			expected: baseTime,
			hasError: false,
		},
		{
			name:     "1 millisecond (00000001)",
			input:    "00000001",
			expected: baseTime.Add(1 * time.Millisecond),
			hasError: false,
		},
		{
			name:     "35 milliseconds (0000000Z)",
			input:    "0000000Z",
			expected: baseTime.Add(35 * time.Millisecond),
			hasError: false,
		},
		{
			name:     "36 milliseconds (00000010)",
			input:    "00000010",
			expected: baseTime.Add(36 * time.Millisecond),
			hasError: false,
		},
		{
			name:     "1 second (000000RS)",
			input:    "000000RS",
			expected: baseTime.Add(1 * time.Second),
			hasError: false,
		},
		{
			name:     "invalid length (7 chars)",
			input:    "1234567",
			expected: time.Time{},
			hasError: true,
		},
		{
			name:     "invalid length (9 chars)",
			input:    "123456789",
			expected: time.Time{},
			hasError: true,
		},
		{
			name:     "invalid character",
			input:    "0000000#", // 不在charset中
			expected: time.Time{},
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Time36ToTime(tc.input)

			if tc.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !result.Equal(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestTimeConversionRoundTrip(t *testing.T) {
	// 测试一些随机时间的往返转换
	testTimes := []time.Time{
		baseTime,
		baseTime.Add(1234567 * time.Millisecond),
		baseTime.Add(987654321 * time.Millisecond),
		time.Date(2023, time.October, 5, 14, 30, 45, 123456789, time.UTC),
		time.Date(2025, time.December, 31, 23, 59, 59, 999999999, time.UTC),
	}

	for i, tTime := range testTimes {
		t.Run(fmt.Sprintf("round trip test %d", i), func(t *testing.T) {
			time36Str := TimeToTime36(tTime)
			convertedTime, err := Time36ToTime(time36Str)

			if err != nil {
				t.Fatalf("conversion error: %v", err)
			}

			// 检查转换回来的时间是否在1毫秒内（考虑可能的精度问题）
			diff := convertedTime.Sub(tTime)
			if diff < -1*time.Millisecond || diff > 1*time.Millisecond {
				t.Errorf("round trip failed: original %v, converted %v, diff %v",
					tTime, convertedTime, diff)
			}
		})
	}
}
