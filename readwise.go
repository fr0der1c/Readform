package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// startSaver starts saver thread.
func startSaver() {
	go saver()
}

func saver() {
	logger.Info("[Readwise] Saver started running...")
	for {
		articles, err := findArticle(nil, false, true, true)
		if err != nil {
			logger.Errorf("[Readwise] findArticle failed: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		if len(articles) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, article := range articles {
			i := 0
			success := false
			for i <= 3 {
				time.Sleep(3 * time.Second)
				token := currentConf.ReadwiseToken
				if token == "" {
					logger.Errorf("[Readwise] Readwise token is empty, cannot send article to readwise.")
					time.Sleep(10 * time.Second)
					continue
				}

				htmlContent, err := readLocalHTMLFile(article.ActualURL)
				if err != nil {
					logger.Errorf("[Readwise] readLocalHTMLFile for URL %v failed: %v", article.URL, err)
					break
				}

				statusCode, content, err := sendToReadwiseReader(article.URL, htmlContent)
				if err != nil {
					logger.Errorf("[Readwise] sendToReadwiseReader failed, statusCode=%v, respContent=%v", statusCode, content)
					i++
				} else {
					logger.Infof("[Readwise] %d Save %s success: %s\n", statusCode, article.URL, content)
					success = true
					err := markURLAsSaved(article.URL, article.Agent, content)
					if err != nil {
						logger.Errorf("[Readwise] markURLAsSaved for URL %s failed: %v", article.URL, err)
					}
					break
				}
			}
			if !success {
				logger.Warnf("[Readwise] save URL %s failed. Will be retried later.", article.URL)
			}
		}
	}
}

func sendToReadwiseReader(url, htmlContent string) (statusCode int, respContent string, err error) {
	type payload struct {
		URL             string `json:"url"`
		HTML            string `json:"html"`
		ShouldCleanHTML bool   `json:"should_clean_html"`
		Location        string `json:"location"`
		SavedUsing      string `json:"saved_using"`
	}

	data := payload{
		URL:             url,
		HTML:            htmlContent,
		ShouldCleanHTML: true,
		Location:        currentConf.ReaderLocation,
		SavedUsing:      "Readform",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, "", err
	}

	req, err := http.NewRequest("POST", "https://readwise.io/api/v3/save/", bytes.NewReader(jsonData))
	if err != nil {
		return 0, "", err
	}

	req.Header.Add("Authorization", "Token "+currentConf.ReadwiseToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	bodyContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("read resp body failed: %w", err)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return resp.StatusCode, string(bodyContent), nil
	}
	return resp.StatusCode, string(bodyContent), fmt.Errorf("got non-success status code")
}
