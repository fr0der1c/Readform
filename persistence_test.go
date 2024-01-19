package main

import (
	"fmt"
	"testing"
)

func TestPersistence(*testing.T) {
	initDB()
	addArticle("http://example.com", "agent1", "")
	markURLAsSaved("http://example2.com", "agent2", "response2")
	article, err := findArticle([]string{"http://example.com"}, false, false, false)
	if err != nil {
		panic(err)
	}
	fmt.Println(article)
}
