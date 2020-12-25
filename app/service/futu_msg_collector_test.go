package service

import (
	"fmt"
	"testing"
)

func TestFutuCollector_Load(t *testing.T) {
	err := TheFutuCollector.LoadFromFile()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	TheFutuCollector.Analysis()
	fmt.Printf("VALIDATION: %v\n", TheFutuCollector.Validation())

	//for idx, msg := range TheFutuCollector.Msgs {
	//	fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//}

	//err = TheFutuCollector.SaveToFile()
	//if err != nil {
	//	fmt.Printf("ERR: %v\n", err)
	//	return
	//}
}

func TestTheFutuCollectorInit(t *testing.T) {
	var (
		err error
	)

	err = TheFutuCollector.LoadFromFile()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	TheFutuCollector.Start()

	fmt.Printf("TOTAL %d MSGS\n", len(TheFutuCollector.Msgs))

	err = TheFutuCollector.Load()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	fmt.Printf("TOTAL %d MSGS\n", len(TheFutuCollector.Msgs))

	<-make(chan struct{}, 1)

	//for idx, msg := range TheFutuCollector.Msgs {
	//	if strings.Contains(msg.RichText, "目标价") && strings.Contains(msg.RichText, "评级") {
	//		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//	}
	//}
	//
	//for idx, msg := range TheFutuCollector.Msgs {
	//	if strings.Contains(msg.RichText, "PLUG") || strings.Contains(msg.RichText, "普拉格") {
	//		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//	}
	//}
}

func TestTheFutuCollectorKeepRefresh(t *testing.T) {
	TheFutuCollector.AddFilter(NewRateFutuMsgFilter())
	err := TheFutuCollector.Start()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}
	//TheFutuCollector.Analysis()

	//for idx, msg := range TheFutuCollector.Msgs {
	//	if strings.Contains(msg.RichText, "目标价") && strings.Contains(msg.RichText, "评级") {
	//		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//	}
	//}
	//
	//for idx, msg := range TheFutuCollector.Msgs {
	//	if strings.Contains(msg.RichText, "PLUG") || strings.Contains(msg.RichText, "普拉格") {
	//		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//	}
	//}
	<-make(chan struct{}, 1)
}
