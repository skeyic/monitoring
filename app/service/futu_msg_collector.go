package service

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/golang/glog"
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

// If do not look back, just check the new message
// Else, check until reach the init message number

var (
	TheFutuCollector = NewFutuCollector(TheFutuCollectorFileName).
		InitMsgNum(FutuDefaultPageSize)
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
		glog.V(4).Infof("MATCH RULE MSG: %+v\n", msg)
		return true
	}
	return false
}

func (r RateFutuMsgFilter) Alert(msg *FutuMsg) error {
	glog.V(4).Infof("ALERT MSG: %+v\n", msg)
	return utils.SendAlertV2(fmt.Sprintf("Rate "+msg.CreateTime), msg.RichText)
}

type TestFutuMsgFilter struct {
}

func NewTestFutuMsgFilter() TestFutuMsgFilter {
	return TestFutuMsgFilter{}
}

func (r TestFutuMsgFilter) Match(msg *FutuMsg) bool {
	if strings.Contains(msg.RichText, "的") {
		glog.V(4).Infof("MATCH RULE MSG: %+v\n", msg)
		return true
	}
	return false
}

func (r TestFutuMsgFilter) Alert(msg *FutuMsg) error {
	return utils.SendAlertV2(fmt.Sprintf("Test "+msg.CreateTime), msg.RichText)
}

type FutuMsg struct {
	CommentID  int64  `json:"idx"`
	CreateTime string `json:"create_time_str"`
	RichText   string `json:"content"`
}

func (s *FutuMsg) ID() int64 {
	return s.CommentID
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

	lookBack bool

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

func (c *FutuCollector) LookBack(lookBack bool) *FutuCollector {
	c.lookBack = lookBack
	return c
}

func (c *FutuCollector) Start() (err error) {
	return c.Process()
}

func (c *FutuCollector) AddFilter(f FutuMsgFilter) {
	c.filters = append(c.filters, f)
}

func (c *FutuCollector) Process() (err error) {
	//err = TheFutuCollector.LoadFromFile()
	//if err != nil {
	//	glog.V(4).Infof("ERR: %v\n", err)
	//	return
	//}

	//// Start auto save after loading
	//go c.AutoSave()

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
					glog.V(4).Infof("LOAD error: %v at %s\n", c.Load(), a)
					return
				}
				glog.V(4).Infof("Another task is running, %s\n", a)
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
				glog.V(4).Infof("Trigger save at %s\n", a)
				err := c.SaveToFile()
				glog.V(4).Infof("Save at %s, err: %v\n", a, err)
			}(a)
		}
	}
}

func checkDuplicate(msgs FutuMsgs) bool {
	var (
		idxMap = make(map[int64]bool)
		result = true
	)

	for idx, msg := range msgs {
		if _, hit := idxMap[msg.CommentID]; hit {
			glog.V(4).Infof("DUPLICATE RECORD: %v\n", msg)
			result = false
		}
		glog.V(4).Infof("IDX: %d, MSG: %v", idx, msg)
		idxMap[msg.CommentID] = true
	}
	return result
}

func (c *FutuCollector) Validation() (result bool) {
	c.msgLock.RLock()
	defer c.msgLock.RUnlock()
	return checkDuplicate(c.Msgs)
}

func (c *FutuCollector) SaveToFile() (err error) {
	data, _ := json.Marshal(c)
	return utils.SaveToFile(c.fileName, data)
}

func (c *FutuCollector) LoadFromFile() (err error) {
	// We do not want to save & load again
	return

	data, err := utils.ReadFromFile(c.fileName)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(4).Infof("LoadFromFile: No such file\n")
			return nil
		}
		return
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return
	}
	sort.Sort(c.Msgs)
	glog.V(4).Infof("LoadFromFile: TOTAL %d MSGS\n", len(c.Msgs))
	return
}

func (c *FutuCollector) GetMsgs(page, pageSize int) (msgs []*FutuMsg, err error) {
	var (
		url = fmt.Sprintf(FutuBaseURL, page, pageSize)
	)

	rCode, rBody, rErr := utils.SendRequest(http.MethodGet, url, nil)
	if rErr != nil {
		glog.V(4).Infof("HTTP ERROR: %v, CODE: %d, BODY: %s\n", rErr, rCode, rBody)
		return
	}

	msgSource, _, _, err := jsonparser.Get([]byte(rBody), "data", "list")
	if err != nil {
		glog.V(4).Infof("Get ERR: %v\n", err)
		return
	}

	err = json.Unmarshal(msgSource, &msgs)
	if err != nil {
		glog.V(4).Infof("Unmarshal ERR: %v\n", err)
		return
	}

	for _, msg := range msgs {
		//glog.V(4).Infof("IDX: %d, MSG: %+v\n", idx, msg)
		msg.AutoMigrate()
	}

	return msgs, err
}

func (c *FutuCollector) MergeMsgs(sourceMsgs, newMsgs []*FutuMsg) (lastMsgs []*FutuMsg) {
	// We know the msg list are descend

	var (
		sourceObjects, newObjects []utils.ToMergeObject
	)

	for _, msg := range sourceMsgs {
		sourceObjects = append(sourceObjects, msg)
	}

	for _, msg := range newMsgs {
		newObjects = append(newObjects, msg)
	}

	lastObjects := utils.MergeDescendObjects(sourceObjects, newObjects)

	for _, msg := range lastObjects {
		lastMsgs = append(lastMsgs, msg.(*FutuMsg))
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
		glog.V(4).Infof("DATE: %s, NUM: %d\n", key, len(value))
	}
}

func (c *FutuCollector) ApplyFilter(msgsToAnalysis []*FutuMsg) {
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
		i              = 0
		msgsBeforeLoad []*FutuMsg
	)

	c.msgLock.RLock()
	msgsBeforeLoad = c.Msgs
	c.msgLock.RUnlock()

	var (
		initial = len(msgsBeforeLoad) == 0
	)

	for {
		msgsThisRound, err := c.GetMsgs(i, FutuDefaultPageSize)
		if err != nil {
			return err
		}
		msgsLengthBeforeMerge := len(msgsBeforeLoad)
		msgsBeforeLoad = c.MergeMsgs(msgsBeforeLoad, msgsThisRound)
		//if !checkDuplicate(msgsBeforeLoad) {
		//	glog.V(4).Infof("FATAL, WE HAVE DUPLICATE RECORD\n")
		//	return nil
		//}
		msgsLengthAfterMerge := len(msgsBeforeLoad)

		c.ApplyFilter(msgsBeforeLoad[:msgsLengthAfterMerge-msgsLengthBeforeMerge])

		c.msgLock.Lock()
		c.Msgs = msgsBeforeLoad
		glog.V(4).Infof("Load more data, current: %d\n", msgsLengthAfterMerge)
		c.msgLock.Unlock()

		glog.V(4).Infof("INIT: %v, CURRENT: %d, BEFORE: %d", initial, msgsLengthAfterMerge, msgsLengthBeforeMerge)

		if initial && msgsLengthAfterMerge >= c.initMsgNum {
			glog.V(4).Infof("Reach the max init msg num, initMsgNum: %d, current: %d", c.initMsgNum, msgsLengthAfterMerge)
			break
		}

		if !initial && msgsLengthAfterMerge-msgsLengthBeforeMerge < FutuDefaultPageSize {
			glog.V(4).Infof("Catch up the msgs, current: %d, previous: %d", msgsLengthAfterMerge, msgsLengthBeforeMerge)
			break
		}

		i++
	}

	//glog.V(4).Infof("Validation after load to check duplicate records: %v\n", c.Validation())

	return nil
}
