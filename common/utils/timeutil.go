package utils

import (
	"strconv"
	"time"
)

func StrToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func CalcBirth2Age(birth string) int {
	birth = Time2StrDay(birth)
	if birth == "" {
		return 0
	}
	birthDate, err := time.Parse("2006-01-02", birth)
	if err != nil {
		return 0
	}

	// Get current date
	now := time.Now()

	// Calculate age
	age := now.Year() - birthDate.Year()

	// Adjust age if birthday hasn't occurred this year yet
	if now.Month() < birthDate.Month() || (now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}

	return age
}

func Time2StrDay(birth string) string {
	if birth == "" {
		return ""
	}
	if len(birth) >= 10 {
		if birthDate, err := time.Parse("2006-01-02", birth[:10]); err == nil {
			return birthDate.Format("2006-01-02")
		}
	}
	if len(birth) == 8 {
		if birthDate, err := time.Parse("20060102", birth); err == nil {
			return birthDate.Format("2006-01-02")
		}
	}
	return ""
}
