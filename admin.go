package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

// startAdmin 启动一个简单的管理面板，提供配置查看与修改接口，并托管静态页面
func startAdmin(addr string) {
    mux := http.NewServeMux()

    // 配置 API - 必须先注册 API 路由，确保 /admin/* 不被静态文件覆盖
    mux.HandleFunc("/admin/config", requireAuth(handleConfig))
    mux.HandleFunc("/admin/backups", requireAuth(handleBackups))
    mux.HandleFunc("/admin/backups/restore", requireAuth(handleBackupRestore))
    mux.HandleFunc("/admin/health", requireAuth(handleHealth))
    mux.HandleFunc("/admin/logs", requireAuth(handleLogs))
    mux.HandleFunc("/admin/audit", requireAuth(handleAudit))
    // Prometheus scrape endpoint (optional, no auth)
    mux.HandleFunc("/metrics", handleMetrics)
    mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "openapi.yaml")
    })

    // 静态文件（管理页面）
    mux.Handle("/", http.FileServer(http.Dir("./static")))

    log.Println("管理面板已启动：", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        log.Println("管理面板服务器出错：", err)
    }
}

func requireAuth(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        token := ""
        if strings.HasPrefix(auth, "Bearer ") {
            token = strings.TrimPrefix(auth, "Bearer ")
        }

        ip := getRequestIP(r)

        // Check token
        tokenOK := cfg.Runtime.AdminToken != "" && token == cfg.Runtime.AdminToken

        // Check IP whitelist
        ipOK := isIPWhitelisted(ip)

        if !tokenOK && !ipOK {
            writeAudit("unknown", "auth_fail", ip, "invalid token or not whitelisted")
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        // Rate limiting (skip for whitelisted IPs)
        if !ipOK && cfg.Runtime.AdminRateLimitPerMin > 0 {
            if limited := checkRateLimit(ip, cfg.Runtime.AdminRateLimitPerMin); limited {
                writeAudit("unknown", "rate_limited", ip, r.URL.Path)
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }
        }

        // Set user for audit: admin if tokenOK else ip
        if tokenOK {
            // continue
        }

        h(w, r)
    }
}

var (
    adminRateMu sync.Mutex
    adminRateMap = make(map[string][]time.Time)
)

func checkRateLimit(ip string, perMin int) bool {
    adminRateMu.Lock()
    defer adminRateMu.Unlock()
    now := time.Now()
    cutoff := now.Add(-1 * time.Minute)
    arr := adminRateMap[ip]
    var recent []time.Time
    for _, t := range arr {
        if t.After(cutoff) {
            recent = append(recent, t)
        }
    }
    if len(recent) >= perMin {
        return true
    }
    recent = append(recent, now)
    adminRateMap[ip] = recent
    return false
}

func getRequestIP(r *http.Request) string {
    // X-Forwarded-For 优先
    if x := r.Header.Get("X-Forwarded-For"); x != "" {
        parts := strings.Split(x, ",")
        if len(parts) > 0 {
            return strings.TrimSpace(parts[0])
        }
    }
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}

func isIPWhitelisted(ip string) bool {
    if len(cfg.Runtime.AdminIPWhitelist) == 0 {
        return false
    }
    for _, e := range cfg.Runtime.AdminIPWhitelist {
        if e == "*" || e == ip {
            return true
        }
    }
    return false
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        w.Header().Set("Content-Type", "application/json")
        // 返回掩码后的配置副本
        masked := cfg
        if masked.Cloudflare.Token != "" {
            masked.Cloudflare.Token = "*****"
        }
        if err := json.NewEncoder(w).Encode(masked); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
        writeAudit("admin", "config_get", r.RemoteAddr, "")
        return
    case http.MethodPost:
        data, err := ioutil.ReadAll(r.Body)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        var newCfg Config
        if err := json.Unmarshal(data, &newCfg); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        // 覆盖内存配置并持久化（saveConfig 返回备份路径）
        cfg = newCfg
        backup, err := saveConfig()
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        writeAudit("admin", "config_update", r.RemoteAddr, backup)
        w.WriteHeader(http.StatusNoContent)
        return
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
}

func handleBackups(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        dir := filepath.Dir(configPath)
        backupsDir := filepath.Join(dir, "backups")
        files, err := ioutil.ReadDir(backupsDir)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        var names []string
        for _, fi := range files {
            if fi.IsDir() {
                continue
            }
            names = append(names, fi.Name())
        }
        sort.Sort(sort.Reverse(sort.StringSlice(names)))
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(names)
        writeAudit("admin", "backups_list", r.RemoteAddr, "")
        return
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
}

func handleBackupRestore(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    data, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    var req struct{
        File string `json:"file"`
    }
    if err := json.Unmarshal(data, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if req.File == "" {
        http.Error(w, "missing file", http.StatusBadRequest)
        return
    }

    dir := filepath.Dir(configPath)
    backupsDir := filepath.Join(dir, "backups")
    backupPath := filepath.Join(backupsDir, req.File)
    bdata, err := ioutil.ReadFile(backupPath)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    tmp := configPath + ".tmp"
    if err := ioutil.WriteFile(tmp, bdata, 0644); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if err := os.Rename(tmp, configPath); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    // reload into memory
    loadConfig()
    writeAudit("admin", "backup_restore", r.RemoteAddr, req.File)
    w.WriteHeader(http.StatusNoContent)
}

// 健康检查
func handleHealth(w http.ResponseWriter, r *http.Request) {
    type h struct {
        StartTime        string `json:"start_time"`
        UptimeSeconds    int64  `json:"uptime_seconds"`
        LastIP           string `json:"last_ip"`
        LastUpdate       string `json:"last_update"`
        UpdatePeriodSecs int64  `json:"update_period_seconds"`
    }
    uptime := int64(time.Since(startTime).Seconds())
    lastUpdateStr := ""
    if !lastUpdate.IsZero() {
        lastUpdateStr = lastUpdate.UTC().Format(time.RFC3339)
    }
    resp := h{StartTime: startTime.UTC().Format(time.RFC3339), UptimeSeconds: uptime, LastIP: lastIP, LastUpdate: lastUpdateStr, UpdatePeriodSecs: int64(updatePeriod.Seconds())}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
    writeAudit("admin", "health", getRequestIP(r), "")
}

// 日志 tail
func handleLogs(w http.ResponseWriter, r *http.Request) {
    tail := 200
    if t := r.URL.Query().Get("tail"); t != "" {
        if v, err := strconv.Atoi(t); err == nil && v > 0 {
            tail = v
        }
    }
    data, err := ioutil.ReadFile(logFile)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    lines := strings.Split(string(data), "\n")
    if len(lines) > tail {
        lines = lines[len(lines)-tail:]
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(lines)
    writeAudit("admin", "logs_tail", getRequestIP(r), fmt.Sprintf("tail=%d", tail))
}

// 审计日志 tail（解析 JSON 行）
func handleAudit(w http.ResponseWriter, r *http.Request) {
    tail := 200
    if t := r.URL.Query().Get("tail"); t != "" {
        if v, err := strconv.Atoi(t); err == nil && v > 0 {
            tail = v
        }
    }
    data, err := ioutil.ReadFile(auditLogPath)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    lines := strings.Split(strings.TrimSpace(string(data)), "\n")
    if len(lines) > tail {
        lines = lines[len(lines)-tail:]
    }
    var out []map[string]interface{}
    for _, ln := range lines {
        var obj map[string]interface{}
        if err := json.Unmarshal([]byte(ln), &obj); err == nil {
            out = append(out, obj)
        }
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(out)
    writeAudit("admin", "audit_tail", getRequestIP(r), fmt.Sprintf("tail=%d", tail))
}

// Prometheus metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain; version=0.0.4")
    fmt.Fprintf(w, "# HELP icmp_ddns_updates_total Total successful updates\n# TYPE icmp_ddns_updates_total counter\nicmp_ddns_updates_total %d\n", atomic.LoadInt64(&updatesTotal))
    fmt.Fprintf(w, "# HELP icmp_ddns_update_errors_total Total update errors\n# TYPE icmp_ddns_update_errors_total counter\nicmp_ddns_update_errors_total %d\n", atomic.LoadInt64(&updateErrorsTotal))
}

