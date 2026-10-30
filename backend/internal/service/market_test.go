package service

import (
	"testing"

	"lunax/internal/domain"
)

func TestMinBid(t *testing.T) {
	cases := []struct {
		highest domain.Cents
		want    domain.Cents
	}{
		{100, 200},       // 1 -> шаг минимум 1
		{1000, 1100},     // 10 -> 10+1
		{10000, 10500},   // 100 -> 100+5
		{123400, 129600}, // 1234 -> 1234+62
	}
	for _, c := range cases {
		if got := MinBid(c.highest); got != c.want {
			t.Errorf("MinBid(%d) = %d, want %d", c.highest, got, c.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	if got := formatNumber("88800000000"); got != "+888 0000 0000" {
		t.Errorf("got %q", got)
	}
	if got := formatNumber("79991234567"); got != "+79991234567" {
		t.Errorf("got %q", got)
	}
}
