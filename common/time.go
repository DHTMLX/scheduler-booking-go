package common

import (
	"fmt"
)

type JTime int

func NewJTime(t int) JTime {
	return JTime(t)
}

func (d *JTime) Get() int { return int(*d) }

func (d *JTime) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", m2t(int(*d)))), nil
}

func m2t(m int) string {
	hours := m / 60
	minutes := m % 60
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}
