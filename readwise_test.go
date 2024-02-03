package main

import "testing"

func TestReadwise(t *testing.T) {
	initConf()

	htmlContent := ``
	url := "https://theinitium.com/zh-Hans/article/20240129-hongkong-skateboard-culture"
	code, resp, err := sendToReadwiseReader(url, htmlContent)
	t.Logf("code: %v resp: %v err: %v", code, resp, err)
}
