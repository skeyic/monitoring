package service

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/skeyic/monitoring/app/utils"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	FutuBaseURL              = "https://news.futunn.com/main/live-list?page=%d&page_size=%d"
	FutuDefaultPageSize      = 50
	FutuDefaultInitMsgNum    = 10000
	FutuDefaultLoadInterval  = 1 * time.Minute
	TheFutuCollectorFileName = "TheFutuCollector.data.test"
)

var (
	TheFutuCollector = NewFutuCollector(TheFutuCollectorFileName).InitMsgNum(FutuDefaultInitMsgNum)
)

type FutuMsgFilter interface {
	Match(msg *FutuMsg) bool
	Alert(msg *FutuMsg) error
}

type RateFutuMsgFilter struct {
}

func NewRateFutuMsgFilter() RateFutuMsgFilter {
	return RateFutuMsgFilter{}
}

func (r RateFutuMsgFilter) Match(msg *FutuMsg) bool {
	if strings.Contains(msg.RichText, "目标价") && strings.Contains(msg.RichText, "评级") {
		//		fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
		//	}
		return true
	}
	return false
}

func (r RateFutuMsgFilter) Alert(msg *FutuMsg) error {
	return utils.SendAlert(fmt.Sprintf("Rate "+msg.CreateTime), msg.RichText)
}

type FutuMsg struct {
	CommentID  int64  `json:"idx"`
	CreateTime string `json:"create_time_str"`
	RichText   string `json:"content"`
}

func (s *FutuMsg) AutoMigrate() {
	if strings.Contains(s.CreateTime, "-") {
		return
	}
	s.CreateTime = time.Now().Format("2006-01-02") + " " + s.CreateTime
}

type FutuMsgs []*FutuMsg

func (s FutuMsgs) Len() int {
	return len(s)
}

// We need the reversed order
func (s FutuMsgs) Less(i, j int) bool {
	return s[i].CommentID > s[j].CommentID
}

func (s FutuMsgs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type FutuCollector struct {
	fileName   string
	initMsgNum int

	msgLock *sync.RWMutex
	Msgs    FutuMsgs

	filters []FutuMsgFilter
}

func NewFutuCollector(fileName string) *FutuCollector {
	return &FutuCollector{
		fileName: fileName,
		msgLock:  &sync.RWMutex{},
	}
}

func (c *FutuCollector) InitMsgNum(initMsgNum int) *FutuCollector {
	c.initMsgNum = initMsgNum
	return c
}

func (c *FutuCollector) Start() (err error) {
	return c.Process()
}

func (c *FutuCollector) AddFilter(f FutuMsgFilter) {
	c.filters = append(c.filters, f)
}

func (c *FutuCollector) Process() (err error) {
	err = TheFutuCollector.LoadFromFile()
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}

	// Start auto save after loading
	go c.AutoSave()

	var (
		ticker = time.NewTicker(FutuDefaultLoadInterval)
		locker = &utils.AsyncLocker{}
	)
	for {
		select {
		case a := <-ticker.C:
			go func(l *utils.AsyncLocker, a time.Time) {
				if l.TryLock() {
					defer l.Unlock()
					fmt.Printf("LOAD error: %v at %s\n", c.Load(), a)
					return
				}
				fmt.Printf("Another task is running, %s\n", a)
			}(locker, a)
		}
	}

}

func (c *FutuCollector) AutoSave() {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case a := <-ticker.C:
			go func(t time.Time) {
				fmt.Printf("Trigger save at %s\n", a)
				err := c.SaveToFile()
				fmt.Printf("Save at %s, err: %v\n", a, err)
			}(a)
		}
	}
}

func checkDuplicate(msgs FutuMsgs) bool {
	var (
		idxMap = make(map[int64]bool)
		result = true
	)

	for _, msg := range msgs {
		if _, hit := idxMap[msg.CommentID]; hit {
			fmt.Printf("DUPLICATE RECORD: %v\n", msg)
			result = false
		}
		idxMap[msg.CommentID] = true
	}
	return result
}

func (c *FutuCollector) Validation() (result bool) {
	c.msgLock.RLock()
	msgs := c.Msgs
	c.msgLock.RUnlock()

	return checkDuplicate(msgs)
}

func (c *FutuCollector) SaveToFile() (err error) {
	data, _ := json.Marshal(c)
	return utils.SaveToFile(c.fileName, data)
}

func (c *FutuCollector) LoadFromFile() (err error) {
	data, err := utils.ReadFromFile(c.fileName)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("LoadFromFile: No such file\n")
			return nil
		}
		return
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return
	}
	sort.Sort(c.Msgs)
	fmt.Printf("LoadFromFile: TOTAL %d MSGS\n", len(c.Msgs))
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

	for _, msg := range msgs {
		//fmt.Printf("IDX: %d, MSG: %+v\n", idx, msg)
		msg.AutoMigrate()
	}

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

func (c *FutuCollector) Analysis() {
	var (
		msgsToAnalysis []*FutuMsg
		dateMap        = make(map[string][]*FutuMsg)
	)
	c.msgLock.RLock()
	msgsToAnalysis = c.Msgs
	c.msgLock.RUnlock()

	for _, msg := range msgsToAnalysis {
		dateMap[strings.Fields(msg.CreateTime)[0]] = append(dateMap[strings.Fields(msg.CreateTime)[0]], msg)
	}

	for key, value := range dateMap {
		fmt.Printf("DATE: %s, NUM: %d\n", key, len(value))
	}
}

func (c *FutuCollector) ApplyFilter() {
	var (
		msgsToAnalysis []*FutuMsg
	)
	c.msgLock.RLock()
	msgsToAnalysis = c.Msgs
	c.msgLock.RUnlock()

	for _, msg := range msgsToAnalysis {
		for _, theFilter := range c.filters {
			if theFilter.Match(msg) {
				theFilter.Alert(msg)
			}
		}
	}
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

	var (
		pauseCount    = 0
		maxPauseCount = 30
		initial       = len(msgsBeforeLoad) == 0
	)

	for {
		if len(msgsBeforeLoad) != 0 {
			lastMsgBeforeLoad = msgsBeforeLoad[0]
		}

		msgsThisRound, err := c.GetMsgs(i, FutuDefaultPageSize)
		if err != nil {
			return err
		}
		msgsLengthBeforeMerge := len(msgsBeforeLoad)
		msgsBeforeLoad = c.MergeMsgs(msgsBeforeLoad, msgsThisRound)
		if !checkDuplicate(msgsBeforeLoad) {
			fmt.Printf("FATAL, WE HAVE DUPLICATE RECORD\n")
			return nil
		}
		msgsLengthAfterMerge := len(msgsBeforeLoad)
		if msgsLengthAfterMerge == msgsLengthBeforeMerge {
			pauseCount++
		} else {
			pauseCount = 0
			c.msgLock.Lock()
			c.Msgs = msgsBeforeLoad
			fmt.Printf("Load more data, current: %d\n", msgsLengthAfterMerge)
			c.msgLock.Unlock()
		}

		if pauseCount == maxPauseCount {
			fmt.Printf("No more data could load, current: %d, last message previous round: %v\n", msgsLengthBeforeMerge, lastMsgBeforeLoad)
			break
		}

		if !initial && lastMsgBeforeLoad != nil && msgsThisRound[len(msgsThisRound)-1].CommentID < lastMsgBeforeLoad.CommentID {
			fmt.Printf("Catch up the msgs, eariest this round: %v, last message previous round: %v\n", msgsThisRound[len(msgsThisRound)-1], lastMsgBeforeLoad)
			break
		}

		if initial && msgsLengthAfterMerge >= c.initMsgNum {
			fmt.Printf("Reach the max init msg num, eariest this round: %v\n", msgsThisRound[len(msgsThisRound)-1])
			break
		}
		i++
	}

	fmt.Printf("Validation after load to check duplicate records: %v\n", c.Validation())

	return nil
}
