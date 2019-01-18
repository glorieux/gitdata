package main

import (
	"fmt"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	t.Skip()
	fileChanges, err := gitChangesOverTime("https://github.com/glorieux/gitdata.git")
	if err != nil {
		t.Error(err)
	}

	if len(fileChanges) == 0 {
		t.Error("Should have file changes")
	}
}

func TestMeanDuration(t *testing.T) {
	now := time.Now()
	timeStamps := []time.Time{
		now.Add(-96 * time.Hour),
		now.Add(-72 * time.Hour),
		now.Add(-48 * time.Hour),
		now.Add(-24 * time.Hour),
	}
	mean := meanDuration(timeStamps)
	if mean == 0 {
		t.Error("Should return mean")
	}
	fmt.Println(timeStamps)
	fmt.Println(mean)
	fmt.Println(time.Duration(mean))
	if mean < 24*time.Hour {
		t.Error("Should be at least 24h appart")
	}
}
