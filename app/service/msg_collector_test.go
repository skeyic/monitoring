package service

import (
	"fmt"
	"testing"
)

func TestLoad(t *testing.T) {
	err := TheSinaFinanceCollector.Load(1000)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	for idx, msg := range TheSinaFinanceCollector.Msgs {
		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	}

	err = TheSinaFinanceCollector.SaveToFile()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}
}

func TestInit(t *testing.T) {
	err := TheSinaFinanceCollector.LoadFromFile()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	for idx, msg := range TheSinaFinanceCollector.Msgs {
		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	}
}
