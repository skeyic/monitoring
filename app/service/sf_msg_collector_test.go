package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	err := TheSinaFinanceCollector.Load(100000)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	//for idx, msg := range TheSinaFinanceCollector.Msgs {
	//	fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//}

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

	fmt.Printf("TOTAL %d MSGS\n", len(TheSinaFinanceCollector.Msgs))

	//for idx, msg := range TheSinaFinanceCollector.Msgs {
	//	if strings.Contains(msg.RichText, "目标价") && strings.Contains(msg.RichText, "评级") {
	//		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//	}
	//}

	for idx, msg := range TheSinaFinanceCollector.Msgs {
		if strings.Contains(msg.RichText, "PLUG") || strings.Contains(msg.RichText, "普拉格") {
			fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
		}
	}
}

func TestConvert(t *testing.T) {
	err := TheSinaFinanceCollector.Load(1)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	b, _ := json.Marshal(TheSinaFinanceCollector)
	fmt.Printf("B: %s\n", b)

	m := SinaFinanceCollector{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	fmt.Printf("M: %+v", m)

}
