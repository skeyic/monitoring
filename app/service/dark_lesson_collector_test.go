package service

import (
	"fmt"
	"testing"
)

func TestDLCollector_Load(t *testing.T) {
	err := TheDLCollector.Load()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
	}
	err = TheDLCollector.Save()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
	}
}
