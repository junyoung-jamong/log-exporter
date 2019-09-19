package main

import (
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Log struct {
	Level       string `json:"level"`
	DateTime    int64  `json:"datetime"`
	Description string `json:"description"`
}

type Logs []Log

const LOG_DIRECTORY = "/opt/meta/logs/"
const LAYOUT = "2006-01-02T15:04:05.000Z"

var FILE_LIST = [6]string{"log", "log.1", "log.2", "log.3", "log.4", "log.5"}

func main() {
	r := gin.Default()
	r.GET("/log", GetLogs)

	r.Run(":9101")
}

func GetLogs(c *gin.Context) {
	start := c.DefaultQuery("start", "")
	startTime, err := time.Parse(LAYOUT, start)

	hasRange := true

	if err != nil {
		hasRange = false
	}

	currentDate := time.Now().Format("2006-01-02")
	currentYear := currentDate[0:4]
	currentMonth, _ := strconv.ParseInt(currentDate[5:7], 10, 64)

	var page string

	for _, fileName := range FILE_LIST {
		filePath := LOG_DIRECTORY + fileName

		f, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}

		segment := string(f)
		page = segment + page

		if hasRange {
			pageStartDt := segment[6:11] + "T" + segment[12:24] + "Z"
			month, _ := strconv.ParseInt(segment[6:8], 10, 64)

			if month > currentMonth {
				pageStartDt = currentYear + "-" + pageStartDt
			} else {
				pageStartDt = currentYear + "-" + pageStartDt
			}

			t1, _ := time.Parse(LAYOUT, pageStartDt)

			if t1.Before(startTime) {
				break
			}
		}
	}

	logList := strings.Split(page, "\n")

	logs := *new(Logs)

	for _, line := range logList {
		if strings.HasPrefix(line, "CRIT") || strings.HasPrefix(line, "ERROR") || strings.HasPrefix(line, "WARN") {
			logDt := line[6:11] + "T" + line[12:24] + "Z"
			month, _ := strconv.ParseInt(line[6:8], 10, 64)

			if month > currentMonth {
				logDt = currentYear + "-" + logDt
			} else {
				logDt = currentYear + "-" + logDt
			}

			t1, _ := time.Parse(LAYOUT, logDt)

			if !hasRange || t1.After(startTime) {
				log := &Log{
					Level:       strings.Trim(line[0:5], " "),
					DateTime:    t1.Unix(),
					Description: line[26:],
				}

				logs = append(logs, *log)
			}
		}
	}

	c.JSON(200, gin.H{
		"result": true,
		"data":   logs,
		"count":  len(logs),
	})
}