package main

// refer
// https://en.wikipedia.org/wiki/Real_Time_Streaming_Protocol
// https://tools.ietf.org/html/rfc2326

// go lang test
// https://tour.golang.org/flowcontrol/1

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
    TaskEvent_CancelingRestart = "canceling_restart"
    TaskEvent_Restarting = "restarting"
)

type Task struct {
    timestamp string
    alloc_id string
    service string
    task string
}

type Service struct {
    Task
    time_limit int
}

type TaskEvent struct {
    Task
    event string
}

func matchUnhealthy(line string) (bool, TaskEvent) {
    var taskEvt TaskEvent
    var time_limit int

    taskEvt.event = TaskEvent_Unhealthy
    isMatch := strings.Contains(line, "consul.health: check became unhealthy.")        
    if isMatch {
        format := "%s [DEBUG] consul.health: check became unhealthy. Will restart if check doesn't become healthy: " +
                        "alloc_id=%s check=\"service: %s check\" task=%s time_limit=%ds"

        count, err := fmt.Sscanf(line, format, &taskEvt.timestamp, &taskEvt.alloc_id, &taskEvt.service, &taskEvt.task, &time_limit)
        if count != 5 || err != nil { 
            return false, taskEvt
        }
        return isMatch, taskEvt
    }

    return false, taskEvt
}

func matchCancelingRestart(line string) (bool, TaskEvent) {
    var taskEvt TaskEvent

    taskEvt.event = TaskEvent_CancelingRestart
    isMatch := strings.Contains(line, "consul.health: canceling restart because check became healthy")        
    if isMatch {
        format := "%s [DEBUG] consul.health: canceling restart because check became healthy: " +
                        "alloc_id=%s check=\"service: %s check\" task=%s"

        count, err := fmt.Sscanf(line, format, &taskEvt.timestamp, &taskEvt.alloc_id, &taskEvt.service, &taskEvt.task)
        if count != 4 || err != nil { 
            return false, taskEvt
        }
        return isMatch, taskEvt
    }

    return false, taskEvt
}

func matchRestarting(line string) (bool, TaskEvent) {
    var taskEvt TaskEvent

    taskEvt.event = TaskEvent_Restarting
    isMatch := strings.Contains(line, "consul.health: restarting due to unhealthy check")        
    if isMatch {
        format := "%s [DEBUG] consul.health: restarting due to unhealthy check: " +
                        "alloc_id=%s check=\"service: %s check\" task=%s"

        count, err := fmt.Sscanf(line, format, &taskEvt.timestamp, &taskEvt.alloc_id, &taskEvt.service, &taskEvt.task)
        if count != 4 || err != nil { 
            return false, taskEvt
        }
        return isMatch, taskEvt
    }

    return false, taskEvt
}

func parse(absPath string, taskMap map[string][]TaskEvent) (int, error) {
    fmt.Printf("parse(\"%s\")\n", absPath)

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
            taskMap[taskEvt.task] = append(taskMap[taskEvt.task], taskEvt)
            continue
        }

        // canceling restart
        ret, taskEvt = matchCancelingRestart(line)
        if ret {
            taskMap[taskEvt.task] = append(taskMap[taskEvt.task], taskEvt)
            continue
        }

        // restarting
        ret, taskEvt = matchRestarting(line)
        if ret {
            taskMap[taskEvt.task] = append(taskMap[taskEvt.task], taskEvt)
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
        ret = -1
    }

    taskMap := make(map[string][]TaskEvent)

    // for each parse file
    for item := logFileQueue.Front(); item != nil; item = item.Next() {
        retp, err := parse(item.Value.(string), taskMap)
        if retp != 0 {
            fmt.Printf("parse(\"%s\") occur error: %s\n", item.Value.(string), err)
        }
    }

    // draw
    for taskName, taskEventSlice := range taskMap {
        for index, taskEvt := range taskEventSlice {
            fmt.Printf("%s>%d, %s, %s\n", taskName, index, taskEvt.timestamp, taskEvt.event)
        }
    }

    return ret
}

func main() {
    path := flag.String("p", "./", "path")
    flag.Parse()

    fmt.Printf("ncProbe path is %s\n", *path)

    ret := probe(*path)
    if ret == 0 {
        time.Sleep(time.Duration(2) * time.Second)
    } else {
        fmt.Println("probe failed: ", ret)
    }

    os.Exit(ret)
}