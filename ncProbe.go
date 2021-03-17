package main

import (
    "bufio"
    "path"
    "path/filepath"
    "os"
    "flag"
    "fmt"
    "time"
    "container/list"
    "strings"
)

const (
    TaskEvent_Unhealthy = "unhealthy"
    TaskEvent_CancelingRestart = "healthy"
    TaskEvent_Restarting = "restarting"
)

type Task struct {
    timestamp string
    alloc_id string
    service string
    task string
}

type TaskEvent struct {
    Task
    event string
    eventType int
}

func timestampFormat(timestamp string) string {
    layout := "2006-01-02T15:04:05.000+0800" 
    dt, err := time.Parse(layout, timestamp)
    if err == nil {
        td := dt.Format("2006-01-02 15:04:05")
        return td
    } else {
        return "1970-01-01 00:00:00"
    }
}

func matchUnhealthy(line string) (bool, TaskEvent) {
    var taskEvt TaskEvent
    var time_limit int

    taskEvt.event = TaskEvent_Unhealthy
    taskEvt.eventType = 1
    isMatch := strings.Contains(line, "consul.health: check became unhealthy.")        
    if isMatch {
        format := "%s [DEBUG] consul.health: check became unhealthy. Will restart if check doesn't become healthy: " +
                        "alloc_id=%s check=\"service: %s check\" task=%s time_limit=%ds"

        count, err := fmt.Sscanf(line, format, &taskEvt.timestamp, &taskEvt.alloc_id, &taskEvt.service, &taskEvt.task, &time_limit)
        if count != 5 || err != nil { 
            return false, taskEvt
        }

        taskEvt.timestamp = timestampFormat(taskEvt.timestamp)
        return isMatch, taskEvt
    }

    return false, taskEvt
}

func matchCancelingRestart(line string) (bool, TaskEvent) {
    var taskEvt TaskEvent

    taskEvt.event = TaskEvent_CancelingRestart
    taskEvt.eventType = 2
    isMatch := strings.Contains(line, "consul.health: canceling restart because check became healthy")        
    if isMatch {
        format := "%s [DEBUG] consul.health: canceling restart because check became healthy: " +
                        "alloc_id=%s check=\"service: %s check\" task=%s"

        count, err := fmt.Sscanf(line, format, &taskEvt.timestamp, &taskEvt.alloc_id, &taskEvt.service, &taskEvt.task)
        if count != 4 || err != nil { 
            return false, taskEvt
        }

        taskEvt.timestamp = timestampFormat(taskEvt.timestamp)
        return isMatch, taskEvt
    }

    return false, taskEvt
}

func matchRestarting(line string) (bool, TaskEvent) {
    var taskEvt TaskEvent

    taskEvt.event = TaskEvent_Restarting
    taskEvt.eventType = 3
    isMatch := strings.Contains(line, "consul.health: restarting due to unhealthy check")        
    if isMatch {
        format := "%s [DEBUG] consul.health: restarting due to unhealthy check: " +
                        "alloc_id=%s check=\"service: %s check\" task=%s"

        count, err := fmt.Sscanf(line, format, &taskEvt.timestamp, &taskEvt.alloc_id, &taskEvt.service, &taskEvt.task)
        if count != 4 || err != nil { 
            return false, taskEvt
        }

        taskEvt.timestamp = timestampFormat(taskEvt.timestamp)
        return isMatch, taskEvt
    }

    return false, taskEvt
}

func parse(absPath string, taskEventMap map[string][]TaskEvent) (int, error) {
    fmt.Printf("parse file: \"%s\"\n", absPath)

    fd, err := os.Open(absPath)
    if err != nil {
        return -1, err
    }
    defer fd.Close()

    scanner := bufio.NewScanner(fd)
    for scanner.Scan() {
        line := scanner.Text()

        // unhealthy
        ret, taskEvt := matchUnhealthy(line)
        if ret {
            taskEventMap[taskEvt.task] = append(taskEventMap[taskEvt.task], taskEvt)
            continue
        }

        // canceling restart
        ret, taskEvt = matchCancelingRestart(line)
        if ret {
            taskEventMap[taskEvt.task] = append(taskEventMap[taskEvt.task], taskEvt)
            continue
        }

        // restarting
        ret, taskEvt = matchRestarting(line)
        if ret {
            taskEventMap[taskEvt.task] = append(taskEventMap[taskEvt.task], taskEvt)
            continue
        }        
    }
    if err := scanner.Err(); err != nil {
        return -2, err
    }

    return 0, nil
}

func probe(workFolder string) int {
    ret := 0
    logFileQueue := list.New()

    currentPath, err := os.Getwd()
    if err != nil {
        fmt.Println(err)
        return -1
    }

    // collect log file
    err = filepath.Walk(workFolder, func(relativePath string, file os.FileInfo, err error) error {
        if file == nil {
            return err
        }
        if file.IsDir() {
            return nil
        }

        suffix := strings.ToUpper(path.Ext(file.Name()))
        // *.log
        if suffix == ".LOG" {
            logFileQueue.PushBack(path.Join(currentPath, relativePath))
        }
        return nil
    })
    if err != nil {
        fmt.Printf("filepath.Walk() returned %v\n", err)
        ret = -2
    }

    taskEventMap := make(map[string][]TaskEvent)

    // for each parse file
    for item := logFileQueue.Front(); item != nil; item = item.Next() {
        retp, err := parse(item.Value.(string), taskEventMap)
        if retp != 0 {
            fmt.Printf("parse: \"%s\" occur error: %s\n", item.Value.(string), err)
        }
    }

    // save csv
    for taskName, taskEventSlice := range taskEventMap {
        outputFileName := taskName + ".csv"
        fd, err := os.Create(outputFileName)
        if err != nil {
            continue
        }
        defer fd.Close()

        fd.WriteString("Timestamp, Type, Event, Index\n")
        for index, taskEvt := range taskEventSlice { 
            line := fmt.Sprintf("%s,%d,%s,%d\n", taskEvt.timestamp, taskEvt.eventType, taskEvt.event, index + 1)

            fd.WriteString(line)

            fmt.Printf(line)
        }

        // fmt.Printf("save task:%s event to %s\n", taskName, outputFileName)
    }

    return ret
}

func main() {
    path := flag.String("p", "./", "path")
    flag.Parse()

    fmt.Printf("ncProbe path is \"%s\"\n", *path)

    ret := probe(*path)
    if ret == 0 {
        time.Sleep(time.Duration(1) * time.Second)
    } else {
        fmt.Println("probe failed: ", ret)
    }

    os.Exit(ret)
}