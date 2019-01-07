package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/fatih/color"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

type filesChanges map[string][]time.Time

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println("A git URL must be passed as argument.")
		os.Exit(-1)
	}
	fileChanges, err := gitChangesOverTime(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	var names []string
	var totalChanges int
	for name, change := range fileChanges {
		totalChanges += len(change)
		names = append(names, name)
		sort.Slice(fileChanges[name], func(i, j int) bool {
			return fileChanges[name][i].Before(fileChanges[name][j])
		})
	}
	sort.Strings(names)

	err = makeCSVOutput(names, fileChanges)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func meanDuration(timeStamps []time.Time) time.Duration {
	durations := []time.Duration{}
	for index, timeStamp := range timeStamps {
		if index == len(timeStamps)-1 {
			durations = append(durations, timeStamp.Sub(time.Now()))
		} else {
			durations = append(durations, timeStamp.Sub(timeStamps[index+1]))
		}
	}
	var mean int64
	for _, duration := range durations {
		mean += int64(duration)
	}
	return -time.Duration(mean / int64(len(durations)))
}

func makeCSVOutput(names []string, fileChanges map[string][]time.Time) error {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"File", "Changes", "Mean days between changes", "Days since last change"}); err != nil {
		return err
	}

	for _, name := range names {
		rangeLen := len(fileChanges[name]) - 1
		mean := meanDuration(fileChanges[name])
		meanLast := meanDuration(fileChanges[name][rangeLen:])
		if meanLast < mean {
			color.Set(color.FgRed)
		}
		line := []string{
			name,
			fmt.Sprint(len(fileChanges[name])),
			fmt.Sprint(int64(mean / (24 * time.Hour))),
			fmt.Sprint(int64(meanLast / (24 * time.Hour))),
		}
		if err := w.Write(line); err != nil {
			return err
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return err
	}
	err := ioutil.WriteFile("report.csv", buf.Bytes(), 0644)
	if err != nil {
		return err
	}
	fmt.Println("Wrote report.csv")
	return nil
}

func gitChangesOverTime(gitURL string) (filesChanges, error) {
	r, err := git.Init(memory.NewStorage(), nil)
	if err != nil {
		return nil, err
	}
	remote, err := r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitURL},
	})
	if err != nil {
		return nil, err
	}
	err = remote.Fetch(&git.FetchOptions{Progress: os.Stdout})
	if err != nil {
		return nil, err
	}

	ref, err := r.Reference("refs/remotes/origin/master", false)
	if err != nil {
		return nil, err
	}

	logs, err := r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}
	defer logs.Close()

	type change struct {
		name      string
		timeStamp time.Time
	}

	fileChanges := make(map[string][]time.Time)

	var wg sync.WaitGroup
	var rWg sync.WaitGroup
	changesChan := make(chan change)

	rWg.Add(1)
	go func() {
		defer rWg.Done()
		for change := range changesChan {
			fileChanges[change.name] = append(fileChanges[change.name], change.timeStamp)
		}
		return
	}()

	err = logs.ForEach(func(commit *object.Commit) error {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, _ := commit.Stats()

			for _, stat := range stats {
				changesChan <- change{name: stat.Name, timeStamp: commit.Author.When}
			}
			return
		}()
		return nil
	})
	if err != nil {
		return nil, err
	}

	wg.Wait()
	close(changesChan)
	rWg.Wait()
	return fileChanges, nil
}
