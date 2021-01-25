package utils

import "testing"

func TestSendAlert(t *testing.T) {
	SendAlert("title", "it is 2:20PM")
}

func TestSendAlertV2(t *testing.T) {
	SendAlertV2("title", "it is 4:46PM")
}
