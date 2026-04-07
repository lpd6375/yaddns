package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "time"
)

func shouldUpdate(ip string) bool {
    now := time.Now()
    hits := hitMap[ip]

    var valid []time.Time
    for _, t := range hits {
        if now.Sub(t) < 5*time.Minute {
            valid = append(valid, t)
        }
    }

    valid = append(valid, now)
    hitMap[ip] = valid

    if len(valid) >= 3 {
        delete(hitMap, ip)
        return true
    }

    return false
}

func getCurrentCFIP() string {
    url := fmt.Sprintf(
        "https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s",
        cfg.Cloudflare.ZoneID,
        cfg.Cloudflare.RecordID,
    )

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+cfg.Cloudflare.Token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return ""
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)

    var result map[string]interface{}
    json.Unmarshal(body, &result)

    if res, ok := result["result"].(map[string]interface{}); ok {
        if content, ok := res["content"].(string); ok {
            return content
        }
    }
    return ""
}

func updateCF(ip string) {
    if ip == lastIP && time.Since(lastUpdate) < updatePeriod {
        return
    }

    current := getCurrentCFIP()
    if current == ip {
        log.Println("IP unchanged:", ip)
        return
    }

    url := fmt.Sprintf(
        "https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s",
        cfg.Cloudflare.ZoneID,
        cfg.Cloudflare.RecordID,
    )

    body := map[string]interface{}{
        "type":    "A",
        "name":    cfg.Cloudflare.Name,
        "content": ip,
        "ttl":     cfg.Cloudflare.TTL,
    }

    data, _ := json.Marshal(body)

    req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(data))
    req.Header.Set("Authorization", "Bearer "+cfg.Cloudflare.Token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        msg := "❌ DDNS更新失败: " + err.Error()
        log.Println(msg)
        writeLog(msg)
        notifyWecom("DDNS更新失败", msg)
        return
    }
    defer resp.Body.Close()

    msg := fmt.Sprintf("✅ DDNS更新成功\nIP: %s\n来源: %s", ip, ip)

    log.Println(msg)
    writeLog(msg)

    notifyWecom("DDNS更新成功", msg)

    lastIP = ip
    lastUpdate = time.Now()
}
