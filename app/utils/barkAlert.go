package utils

import (
	"fmt"
	"net/http"
)

func SendAlert(title, content string) error {
	var (
		barkURL = "https://api.day.app/kMHL4X8KSWDWzhZyZY3hgk/%s/%s"
	)

	fmt.Println(SendRequest(http.MethodPost, fmt.Sprintf(barkURL, title, content), nil))

	return nil
}
