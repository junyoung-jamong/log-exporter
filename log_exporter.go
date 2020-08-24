package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
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

const LAYOUT = "2006-01-02T15:04:05.000Z"

var LOG_DIRECTORY *string
var FILE_LIST = [6]string{"log", "log.1", "log.2", "log.3", "log.4", "log.5"}

var IGNORE_LIST []string

func main() {
	parseFlag()

	r := gin.Default()
	r.GET("/ping", PING)
	r.GET("/log", GetLogs)
	r.GET("/reboot", Reboot)
	r.GET("/restart", ReStart)

	r.Run(":9101")
}

func parseFlag() {
	LOG_DIRECTORY = flag.String("log", "/opt/meta/logs/", "Input log file directory")
	ignoreFile := flag.String("ignore", "", "Input ignore file path")
	flag.Parse()

	f, err := ioutil.ReadFile(*ignoreFile)
	if *ignoreFile != "" && err != nil { //파일이 존재하지 않거나 접근에 실패했을 경우
		print("Ignore file error:", err)
	} else {
		ignores := string(f)
		IGNORE_LIST = strings.Split(ignores, "\n")
		fmt.Println("Ignore messages:", len(IGNORE_LIST), IGNORE_LIST)
	}
}

func PING(c *gin.Context) {
	c.JSON(200, gin.H{
		"result":  true,
		"message": "pong",
	})
}

func Reboot(c *gin.Context) {
	cmd := exec.Command("sh", "reboot.sh")
	err := cmd.Start()
	cmd.Wait()

	if err != nil {
		c.JSON(200, gin.H{
			"result":  false,
			"message": err,
		})
	} else {
		c.JSON(200, gin.H{
			"result": true,
		})
	}
}

func ReStart(c *gin.Context) {
	cmd := exec.Command("sh", "restart.sh")
	err := cmd.Start()
	cmd.Wait()

	if err != nil {
		c.JSON(200, gin.H{
			"result":  false,
			"message": err,
		})
	} else {
		c.JSON(200, gin.H{
			"result": true,
		})
	}
}

func GetLogs(c *gin.Context) {
	start := c.DefaultQuery("start", "")

	startInt, err := strconv.ParseInt(start, 10, 64)

	hasRange := true
	startTime := time.Unix(0, 0) //Default Time: 1970-01-01 09:00:00 +0900 KST

	if err != nil {
		hasRange = false
	} else {
		startTime = time.Unix(startInt, 0).UTC()
		fmt.Println("startTime", startTime)
	}

	currentDate := time.Now().Format("2006-01-02")
	currentYear, _ := strconv.ParseInt(currentDate[0:4], 10, 64)
	currentMonth, _ := strconv.ParseInt(currentDate[5:7], 10, 64)

	var page string

	for _, fileName := range FILE_LIST {
		filePath := *LOG_DIRECTORY + fileName

		f, err := ioutil.ReadFile(filePath)
		if err != nil { //파일이 존재하지 않거나 접근에 실패했을 경우
			continue
		}

		segment := string(f)
		page = segment + page

		if hasRange {
			pageStartDt := segment[6:11] + "T" + segment[12:24] + "Z" //Log파일의 첫 번째 Log 일자
			month, _ := strconv.ParseInt(segment[6:8], 10, 64)

			if month > currentMonth {
				pageStartDt = strconv.Itoa(int(currentYear-1)) + "-" + pageStartDt
			} else {
				pageStartDt = strconv.Itoa(int(currentYear)) + "-" + pageStartDt
			}

			t1, _ := time.Parse(LAYOUT, pageStartDt)

			if t1.Before(startTime) {
				break
			}
		}
	}

	logList := strings.Split(page, "\n")
	logs := *new(Logs)

	//라인별로 로그 여부 판단
	for _, line := range logList {
		if strings.HasPrefix(line, "CRIT") || strings.HasPrefix(line, "ERROR") || strings.HasPrefix(line, "WARN") {
			logLevel := strings.Trim(line[0:5], " ")      //Log Level
			logDt := line[6:11] + "T" + line[12:24] + "Z" //Log datetime
			message := line[26:]                          //Log Description

			//로그 제외 목록 체크
			isIgnored := false
			for _, ignore := range IGNORE_LIST {
				if len(ignore) > 0 && strings.Contains(line, ignore) {
					isIgnored = true
					break
				}
			}

			if isIgnored {
				continue
			}

			//로그 날짜 형식 변환
			month, _ := strconv.ParseInt(line[6:8], 10, 64)

			if month > currentMonth {
				logDt = strconv.Itoa(int(currentYear-1)) + "-" + logDt
			} else {
				logDt = strconv.Itoa(int(currentYear)) + "-" + logDt
			}

			t1, _ := time.Parse(LAYOUT, logDt)

			//범위 조건 검사 및 반환 형식화
			if !hasRange || t1.After(startTime) {
				log := &Log{
					Level:       logLevel,
					DateTime:    t1.Unix(),
					Description: message,
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
