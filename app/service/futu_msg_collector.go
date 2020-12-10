package service

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/skeyic/monitoring/app/utils"
	"net/http"
	"sync"
)

const (
	FutuBaseURL              = "https://news.futunn.com/main/live-list?page=%d&page_size=%d"
	FutuDefaultPageSize      = 50
	FutuDefaultInitMsgNum    = 5000
	TheFutuCollectorFileName = "TheFutuCollector.data"
)

var (
	TheFutuCollector = NewFutuCollector(TheFutuCollectorFileName).InitMsgNum(FutuDefaultInitMsgNum)
)

type FutuMsg struct {
	CommentID  int64  `json:"idx"`
	CreateTime string `json:"create_time_str"`
	RichText   string `json:"content"`
}

type FutuMsgs []*FutuMsg

func (s FutuMsgs) Len() int {
	return len(s)
}

func (s FutuMsgs) Less(i, j int) bool {
	return s[i].CommentID < s[j].CommentID
}

func (s FutuMsgs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type FutuCollector struct {
	fileName   string
	initMsgNum int64

	msgLock   *sync.RWMutex
	Msgs      []*FutuMsg
	LatestMsg *FutuMsg
}

func NewFutuCollector(fileName string) *FutuCollector {
	return &FutuCollector{
		fileName: fileName,
		msgLock:  &sync.RWMutex{},
	}
}

func (c *FutuCollector) InitMsgNum(initMsgNum int64) *FutuCollector {
	c.initMsgNum = initMsgNum
	return c
}

func (c *FutuCollector) SaveToFile() (err error) {
	data, _ := json.Marshal(c)
	return utils.SaveToFile(c.fileName, data)
}

func (c *FutuCollector) LoadFromFile() (err error) {
	data, err := utils.ReadFromFile(c.fileName)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return
	}
	return
}

func (c *FutuCollector) GetMsgs(page, pageSize int) (msgs []*FutuMsg, err error) {
	var (
		url = fmt.Sprintf(FutuBaseURL, page, pageSize)
	)

	rCode, rBody, rErr := utils.SendRequest(http.MethodGet, url, nil)
	if rErr != nil {
		fmt.Printf("HTTP ERROR: %v, CODE: %d, BODY: %s\n", rErr, rCode, rBody)
		return
	}

	msgSource, _, _, err := jsonparser.Get([]byte(rBody), "data", "list")
	if err != nil {
		fmt.Printf("Get ERR: %v\n", err)
		return
	}

	err = json.Unmarshal(msgSource, &msgs)
	if err != nil {
		fmt.Printf("Unmarshal ERR: %v\n", err)
		return
	}

	//for idx, msg := range msgs {
	//	fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
	//}

	return msgs, err
}

func (c *FutuCollector) MergeMsgs(sourceMsgs, newMsgs []*FutuMsg) (lastMsgs []*FutuMsg) {
	// We know the msg list are descend

	var (
		newMsgsAnchor, sourceMsgsAnchor = 0, 0
		newMsgsLength, sourceMsgsLength = len(newMsgs), len(sourceMsgs)
	)

	if sourceMsgsLength == 0 {
		return newMsgs
	}
	if newMsgsLength == 0 {
		return sourceMsgs
	}

	if newMsgs[newMsgsLength-1].CommentID > sourceMsgs[0].CommentID {
		return append(sourceMsgs, newMsgs...)
	}

	if newMsgs[newMsgsLength-1].CommentID > sourceMsgs[0].CommentID {
		return append(sourceMsgs, newMsgs...)
	}

	for {
		if newMsgs[newMsgsAnchor].CommentID > sourceMsgs[sourceMsgsAnchor].CommentID {
			lastMsgs = append(lastMsgs, newMsgs[newMsgsAnchor])
			newMsgsAnchor++
		} else if newMsgs[newMsgsAnchor].CommentID < sourceMsgs[sourceMsgsAnchor].CommentID {
			lastMsgs = append(lastMsgs, sourceMsgs[sourceMsgsAnchor])
			sourceMsgsAnchor++
		} else {
			lastMsgs = append(lastMsgs, sourceMsgs[sourceMsgsAnchor])
			sourceMsgsAnchor++
			newMsgsAnchor++
		}

		if newMsgsAnchor == newMsgsLength {
			lastMsgs = append(lastMsgs, sourceMsgs[sourceMsgsAnchor:]...)
			break
		}
		if sourceMsgsAnchor == sourceMsgsLength {
			lastMsgs = append(lastMsgs, newMsgs[newMsgsAnchor:]...)
			break
		}
	}

	return
}

func (c *FutuCollector) Load() (err error) {
	var (
		i                 = 0
		msgsBeforeLoad    []*FutuMsg
		lastMsgBeforeLoad *FutuMsg
	)

	c.msgLock.RLock()
	msgsBeforeLoad = c.Msgs
	c.msgLock.RUnlock()

	if len(msgsBeforeLoad) != 0 {
		lastMsgBeforeLoad = msgsBeforeLoad[len(msgsBeforeLoad)-1]
	}

	var (
		pauseCount    = 0
		maxPauseCount = 30
	)

	for {
		msgsThisRound, err := c.GetMsgs(i, FutuDefaultPageSize)
		if err != nil {
			return err
		}
		msgsLengthBeforeMerge := len(msgsBeforeLoad)
		msgsBeforeLoad = c.MergeMsgs(msgsBeforeLoad, msgsThisRound)
		msgsLengthAfterMerge := len(msgsBeforeLoad)
		fmt.Printf("PAGE: %d, Before %d data, After %d data\n", i, msgsLengthBeforeMerge, msgsLengthAfterMerge)
		if msgsLengthAfterMerge == msgsLengthBeforeMerge {
			pauseCount++
		} else {
			pauseCount = 0
		}

		if pauseCount == maxPauseCount {
			fmt.Printf("No more data could load, current: %d\n", msgsLengthBeforeMerge)
			fmt.Printf("Last message: %v\n", msgsBeforeLoad[msgsLengthAfterMerge-1])
			break
		}

		if lastMsgBeforeLoad != nil && msgsBeforeLoad[msgsLengthAfterMerge-1].CommentID < lastMsgBeforeLoad.CommentID {
			fmt.Printf("%d\n", msgsBeforeLoad[msgsLengthAfterMerge-1].CommentID)
			fmt.Printf("%d\n", lastMsgBeforeLoad.CommentID)
			fmt.Printf("Catch up the msgs\n")
			break
		}
		i++
	}

	c.msgLock.Lock()
	c.Msgs = msgsBeforeLoad
	c.msgLock.Unlock()

	return nil
}
