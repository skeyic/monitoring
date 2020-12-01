package service

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/skeyic/monitoring/app/utils"
	"net/http"
)

const (
	SinaFinanceBaseURL              = "http://zhibo.sina.com.cn/api/zhibo/feed?page=%d&page_size=%d&zhibo_id=152"
	DefaultPageSize                 = 100
	TheSinaFinanceCollectorFileName = "TheSinaFinanceCollector.data"
)

var (
	TheSinaFinanceCollector = &SinaFinanceCollector{
		fileName: TheSinaFinanceCollectorFileName,
	}
)

type SinaFinanceMsg struct {
	CommentID  string `json:"commentid"`
	CreateTime string `json:"create_time"`
	RichText   string `json:"rich_text"`
}

type SinaFinanceMsgs []*SinaFinanceMsg

func (s SinaFinanceMsgs) Len() int {
	return len(s)
}

func (s SinaFinanceMsgs) Less(i, j int) bool {
	return s[i].CreateTime < s[j].CreateTime
}

func (s SinaFinanceMsgs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type SinaFinanceCollector struct {
	Msgs     []*SinaFinanceMsg
	fileName string
}

func (c *SinaFinanceCollector) SaveToFile() (err error) {
	data, _ := json.Marshal(c)
	return utils.SaveToFile(c.fileName, data)
}

func (c *SinaFinanceCollector) LoadFromFile() (err error) {
	var (
		n = &SinaFinanceCollector{}
	)
	data, err := utils.ReadFromFile(c.fileName)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &n)
	if err != nil {
		return
	}

	c.MergeMsgs(c.Msgs, n.Msgs)
	return
}

func (c *SinaFinanceCollector) GetMsgs(page, pageSize int) (msgs []*SinaFinanceMsg, err error) {
	var (
		url = fmt.Sprintf(SinaFinanceBaseURL, page, pageSize)
	)

	rCode, rBody, rErr := utils.SendRequest(http.MethodGet, url, nil)
	if rErr != nil {
		fmt.Printf("HTTP ERROR: %v, CODE: %d, BODY: %s\n", rErr, rCode, rBody)
		return
	}

	msgSource, _, _, err := jsonparser.Get([]byte(rBody), "result", "data", "feed", "list")
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

func (c *SinaFinanceCollector) MergeMsgs(sourceMsgs, newMsgs []*SinaFinanceMsg) (lastMsgs []*SinaFinanceMsg) {
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

	if newMsgs[newMsgsLength-1].CreateTime > sourceMsgs[0].CreateTime {
		return append(sourceMsgs, newMsgs...)
	}

	if newMsgs[newMsgsLength-1].CreateTime > sourceMsgs[0].CreateTime {
		return append(sourceMsgs, newMsgs...)
	}

	for {
		if newMsgs[newMsgsAnchor].CreateTime > sourceMsgs[sourceMsgsAnchor].CreateTime {
			lastMsgs = append(lastMsgs, newMsgs[newMsgsAnchor])
			newMsgsAnchor++
		} else if newMsgs[newMsgsAnchor].CreateTime < sourceMsgs[sourceMsgsAnchor].CreateTime {
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

func (c *SinaFinanceCollector) Load(maxLength int64) (err error) {
	var (
		i = 1
	)

	for {
		currentMsgs, err := c.GetMsgs(i, DefaultPageSize)
		if err != nil {
			return err
		}
		c.Msgs = c.MergeMsgs(c.Msgs, currentMsgs)
		if int64(len(c.Msgs)) > maxLength {
			break
		}
		i++
	}

	return nil
}
