package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

func writeLog(msg string) {
    f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    defer f.Close()
    f.WriteString(time.Now().Format("2006-01-02 15:04:05") + " " + msg + "\n")
}

func notifyWecom(title, content string) {
    if cfg.Notify.WecomWebhook == "" {
        return
    }

    body := map[string]interface{}{
        "msgtype": "text",
        "text": map[string]string{
            "content": fmt.Sprintf("%s\n\n%s", title, content),
        },
    }

    data, _ := json.Marshal(body)
    http.Post(cfg.Notify.WecomWebhook, "application/json", bytes.NewBuffer(data))
}
