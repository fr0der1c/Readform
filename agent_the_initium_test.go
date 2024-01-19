package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestTheInitium(t *testing.T) {
	Convey("Test TheInitium", t, func() {
		initLogger()

		ctx, cancelFunc := GetBrowserCtx()

		agent := WebsiteAgent{iWebsiteAgent: &TheInitium{}}
		err := agent.Init(ctx, cancelFunc, &AgentConf{
			Username: "<fill username here before running test>",
			Password: "<fill password here before running test>",
		})
		So(err, ShouldBeNil)

		// test free articles
		testURL := "https://theinitium.com/zh-Hans/article/20240114-audio-taiwan-election-three-woman-story"
		url, content, err := agent.GetPageContent(testURL)
		So(err, ShouldBeNil)
		So(url == testURL || url == "https://theinitium.com/article/20240114-audio-taiwan-election-three-woman-story", ShouldBeTrue)
		So(len(content), ShouldBeGreaterThan, 1000)

		time.Sleep(3 * time.Second) // last tab should be closed

		// test paid articles
		testURL = "https://theinitium.com/zh-Hans/article/20240111-opinion-china-tourism"
		url, content, err = agent.GetPageContent(testURL)
		So(err, ShouldBeNil)
		So(url, ShouldEqual, testURL)
		So(len(content), ShouldBeGreaterThan, 1000)
	})
}
