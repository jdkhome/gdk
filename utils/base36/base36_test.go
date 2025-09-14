package base36

import (
	"fmt"
	"math"
	"testing"
)

func TestUint64ToBase36(t *testing.T) {
	testCases := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "A"},
		{35, "Z"},
		{36, "10"},
		{37, "11"},
		{1295, "ZZ"},
		{1296, "100"},
		{math.MaxUint64, "3W5E11264SGSF"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := Uint64ToBase36(tc.input)
			if result != tc.expected {
				t.Errorf("转换失败：输入 %d，期望 %s，得到 %s", tc.input, tc.expected, result)
			}
		})
	}
}

func TestBase36ToUint64(t *testing.T) {
	testCases := []struct {
		input    string
		expected uint64
		isValid  bool
	}{
		{"0", 0, true},
		{"1", 1, true},
		{"A", 10, true},
		{"Z", 35, true},
		{"10", 36, true},
		{"11", 37, true},
		{"ZZ", 1295, true},
		{"100", 1296, true},
		{"3W5E11264SGSF", math.MaxUint64, true},
		{"3W5E11264SGSG", 0, false},     // 溢出
		{"invalid!", 0, false},          // 包含非法字符!
		{"10000000000000000", 0, false}, // 溢出
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := Base36ToUint64(tc.input)
			if tc.isValid {
				if err != nil {
					t.Errorf("意外错误：%v", err)
				}
				if result != tc.expected {
					t.Errorf("转换失败：输入 %s，期望 %d，得到 %d", tc.input, tc.expected, result)
				}
			} else {
				if err == nil {
					t.Errorf("期望错误，但转换成功：结果 %d", result)
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	testValues := []uint64{
		0, 1, 10, 35, 36, 1295, 1296,
		math.MaxUint64 / 2, math.MaxUint64,
	}

	for _, value := range testValues {
		t.Run(fmt.Sprintf("%d", value), func(t *testing.T) {
			// 正向转换
			str := Uint64ToBase36(value)

			// 反向转换
			result, err := Base36ToUint64(str)
			if err != nil {
				t.Fatalf("反向转换失败：%v", err)
			}

			if result != value {
				t.Errorf("往返转换失败：原始值 %d，转换后 %d", value, result)
			}
		})
	}
}

func TestBase63(t *testing.T) {
	t.Logf(Uint64ToBase36(86400000))
	num, err := Base36ToUint64("0000000g")

	t.Logf("num:%v err:%v", num, err)
}
