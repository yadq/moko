package main

import (
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWatching(t *testing.T) {
	result := map[string]bool{
		"write": false,
	}

	nw := NewFileWatcher()

	Convey("write watching", t, func() {
		writer := func() error {
			result["write"] = true
			return nil
		}
		os.WriteFile("test", []byte("abcde"), 0644)
		nw.Watch("test", writer)
		time.Sleep(1 * time.Millisecond)
		fp, _ := os.OpenFile("test", os.O_RDWR, 0644)
		fp.WriteString("defgh")
		fp.Close()
		defer os.Remove("test")
		time.Sleep(1 * time.Millisecond)
		So(result["write"], ShouldBeTrue)

		result["write"] = false
		nw.Stop()
		fp, _ = os.OpenFile("test", os.O_RDWR, 0644)
		fp.WriteString("klkhmn")
		fp.Close()
		time.Sleep(1 * time.Millisecond)
		So(result["write"], ShouldBeFalse)
	})
}
