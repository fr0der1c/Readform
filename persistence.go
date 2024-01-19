package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Article struct {
	URL             string `gorm:"type:varchar(2048);primaryKey"`
	Agent           string `gorm:"type:varchar(128)"`
	SavedToReadwise bool   `gorm:"type:boolean"`
	SaveTime        string `gorm:"type:varchar(128)"`
	ReadwiseResp    string `gorm:"type:varchar(1024)"`
	// Content         string `gorm:"type:varchar"`
	ActualURL   string
	PublishTime time.Time `gorm:"type:datetime"`
	CreateTime  time.Time `gorm:"type:datetime;autoCreateTime"`
	UpdateTime  time.Time `gorm:"type:datetime;autoUpdateTime"`
}

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("data/readform.db"), &gorm.Config{NamingStrategy: schema.NamingStrategy{
		SingularTable: true,
	}})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&Article{})
	if err != nil {
		panic(err)
	}
}

func addArticle(url string, agent string, actualURL string) error {
	var articles []*Article
	err := db.Find(&articles, "url = ?", url).Error
	if err != nil {
		return err
	}
	if len(articles) == 0 {
		// Article does not exist, create a new one
		article := Article{
			URL:       url,
			ActualURL: actualURL,
			Agent:     agent,
		}
		return db.Create(&article).Error
	} else {
		// Article exists, update it
		return db.Model(&articles[0]).Updates(Article{Agent: agent}).Error
	}
}

func markURLAsSaved(url string, agent string, resp string) error {
	var article Article
	if err := db.First(&article, "url = ?", url).Error; err != nil {
		// Article does not exist, create a new one
		article = Article{
			URL:             url,
			Agent:           agent,
			SavedToReadwise: true,
			SaveTime:        time.Now().Format("2006-01-02 15:04:05"),
			ReadwiseResp:    resp,
		}
		return db.Create(&article).Error
	} else {
		// Article exists, update it
		return db.Model(&article).Updates(Article{
			SavedToReadwise: true,
			SaveTime:        time.Now().Format("2006-01-02 15:04:05"),
			Agent:           agent,
			ReadwiseResp:    resp,
		}).Error
	}
}

// findArticle finds article from database. Legacy versions of Readform does not have ActualURL field,
// so hasActualURL=true can filter out items created by legacy version.
func findArticle(urlList []string, onlySaved bool, onlyNotSaved bool, hasActualURL bool) ([]Article, error) {
	var articles []Article
	tx := db
	if urlList != nil {
		tx = tx.Where("url IN (?)", urlList)
	}
	if onlySaved {
		tx = tx.Where("saved_to_readwise = ?", true)
	}
	if onlyNotSaved {
		tx = tx.Where("saved_to_readwise = ?", false)
	}
	if hasActualURL {
		tx = tx.Where("actual_url != ''")
	}

	if err := tx.Find(&articles).Error; err != nil {
		return nil, err
	}

	return articles, nil
}

// filterOldURLs filter out saved URLs, returning unsaved URLs.
func filterOldURLs(urls []string) ([]string, error) {
	articles, err := findArticle(urls, true, false, false)
	if err != nil {
		return nil, fmt.Errorf("findArticle failed: %w", err)
	}
	existURLs := make(map[string]struct{}, len(articles))
	for _, a := range articles {
		existURLs[a.URL] = struct{}{}
	}

	var unsavedURLs []string
	for _, url := range urls {
		if _, exist := existURLs[url]; !exist {
			unsavedURLs = append(unsavedURLs, url)
		}
	}
	unsavedURLs = UniqStringSlice(unsavedURLs)
	return unsavedURLs, nil
}

func urlToLocalFilePath(url string) string {
	fileName := strings.ReplaceAll(url, "/", "_")
	fileName = strings.ReplaceAll(fileName, ":", "")
	filePath := "data/html/" + fileName + ".html"
	return filePath
}

// saveHTMLToLocalFile saves URL content to local file.
func saveHTMLToLocalFile(url, htmlContent string) error {
	filePath := urlToLocalFilePath(url)

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory failed: %w", err)
	}

	err := os.WriteFile(filePath, []byte(htmlContent), 0644)
	if err != nil {
		return fmt.Errorf("WriteFile failed: %w", err)
	}
	return nil
}

// readLocalHTMLFile gets URL content from local file.
func readLocalHTMLFile(url string) (string, error) {
	filePath := urlToLocalFilePath(url)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
