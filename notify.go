package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
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
    if !cfg.Notify.EnableWecom || cfg.Notify.WecomWebhook == "" {
        return
    }

    body := map[string]interface{}{
        "msgtype": "text",
        "text": map[string]string{
            "content": fmt.Sprintf("%s\n\n%s", title, content),
        },
    }

    data, _ := json.Marshal(body)
    resp, err := http.Post(cfg.Notify.WecomWebhook, "application/json", bytes.NewBuffer(data))
    if err != nil {
        writeLog("wecom notify failed: " + err.Error())
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        writeLog(fmt.Sprintf("wecom notify failed: status=%s body=%s", resp.Status, string(b)))
    }
}

func notifyTelegram(title, content string) {
    if !cfg.Notify.EnableTelegram || cfg.Notify.TelegramBotToken == "" || cfg.Notify.TelegramChatID == "" {
        return
    }

    text := fmt.Sprintf("%s\n\n%s", title, content)
    body := map[string]string{
        "chat_id": cfg.Notify.TelegramChatID,
        "text":    text,
    }
    data, _ := json.Marshal(body)
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.Notify.TelegramBotToken)
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
    if err != nil {
        writeLog("telegram notify failed: " + err.Error())
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        writeLog(fmt.Sprintf("telegram notify failed: status=%s body=%s", resp.Status, string(b)))
    }
}

func notifyAll(title, content string) {
    if cfg.Notify.EnableWecom && cfg.Notify.WecomWebhook != "" {
        go notifyWecom(title, content)
    }
    if cfg.Notify.EnableTelegram && cfg.Notify.TelegramBotToken != "" && cfg.Notify.TelegramChatID != "" {
        go notifyTelegram(title, content)
    }
}
