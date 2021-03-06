package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/hpcloud/tail"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type DataStore struct {
	Logs []*LogItem
}

type LogItem struct {
	rawData string
}

type ParsedLog struct {
	url                      string
	method                   string
	timestamp                string
	responseCode             string
	responseTime             string
	genesisTotalResponseTime string
}

func (l ParsedLog) headline() string {
	var headlineItems = make([]string, 0)
	var responseCode string
	urlParts := strings.Split(l.url, "?")
	statusCode, _ := strconv.Atoi(l.responseCode)
	if statusCode < 300 {
		responseCode = color.GreenString(l.responseCode)
	} else if statusCode >= 300 && statusCode < 400 {
		responseCode = color.CyanString(l.responseCode)
	} else if statusCode >= 400 && statusCode < 500 {
		responseCode = color.YellowString(l.responseCode)
	} else {
		responseCode = color.RedString(l.responseCode)
	}
	url := color.WhiteString(urlParts[0])
	headlineItems = append(headlineItems, fmt.Sprintf("%12s %4s %s", responseCode, l.method, url))
	if l.responseTime != "" {
		headlineItems = append(headlineItems, fmt.Sprintf("in %sms", l.responseTime))
	}
	if l.genesisTotalResponseTime != "" {
		headlineItems = append(headlineItems, fmt.Sprintf("api:%sms", l.genesisTotalResponseTime))
	}
	return strings.Join(headlineItems, " ")
}

func (l LogItem) parsedLog() ParsedLog {
	log := new(ParsedLog)
	var startedRegex = regexp.MustCompile(`Started (\w+) "(.*?)" for ([a-zA-Z:0-9\.]*) at (.*)`)
	if startedRegex.MatchString(l.rawData) {
		var matches = startedRegex.FindStringSubmatch(l.rawData)
		log.url = matches[2]
		log.method = matches[1]
		log.timestamp = matches[4]
	}
	var completedRegex = regexp.MustCompile(`Completed\s*(?P<response_code>\w+)\s*(?P<response_description>[\w\s]+) in (?P<response_time>\d+)ms(?:\s\(Genesis: (?P<genesis_total_response_time>\d+\.\d*)ms\))?`)
	if completedRegex.MatchString(l.rawData) {
		var matches = completedRegex.FindStringSubmatch(l.rawData)
		log.responseCode = matches[1]
		log.responseTime = matches[3]
		log.genesisTotalResponseTime = matches[4]
	}
	return *log
}

var dataStore DataStore

var done = make(chan bool)
var logItems = make(chan LogItem)

func main() {
	go parser()
	go echoer()
	<-done
}

func parser() {
	t, err := tail.TailFile(os.Args[1], tail.Config{Follow: true})
	if err != nil {
		panic(err)
	}
	logItem := new(LogItem)
	for line := range t.Lines {
		match, _ := regexp.MatchString("Started ", line.Text)
		if match {
			if logItem.rawData != "" {
				logItems <- *logItem
			}
			logItem = new(LogItem)
			logItem.rawData = line.Text
		} else {
			logItem.rawData += "\n"
			logItem.rawData += line.Text
		}
	}

	done <- true
}

func echoer() {
	for {
		log := <-logItems
		fmt.Println(log.parsedLog().headline())
	}
}
