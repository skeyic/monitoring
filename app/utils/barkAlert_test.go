package utils

import "testing"

func TestSendAlert(t *testing.T) {
	SendAlert("title", "it is 2:20PM")
}
