package main

import "time"

type Config struct {
    Interface string `yaml:"interface" json:"interface"`
    Cloudflare struct {
        Token    string `yaml:"token" json:"token"`
        ZoneID   string `yaml:"zone_id" json:"zone_id"`
        RecordID string `yaml:"record_id" json:"record_id"`
        Name     string `yaml:"name" json:"name"`
        TTL      int    `yaml:"ttl" json:"ttl"`
    } `yaml:"cloudflare" json:"cloudflare"`
    Runtime struct {
        UpdateInterval int `yaml:"update_interval" json:"update_interval"`
        AdminAddr      string `yaml:"admin_addr" json:"admin_addr"`
        AdminToken     string `yaml:"admin_token" json:"admin_token"`
        AdminIPWhitelist []string `yaml:"admin_ip_whitelist" json:"admin_ip_whitelist"`
        AdminRateLimitPerMin int `yaml:"admin_rate_limit_per_min" json:"admin_rate_limit_per_min"`
    } `yaml:"runtime" json:"runtime"`
    Notify struct {
        WecomWebhook string `yaml:"wecom_webhook" json:"wecom_webhook"`
        EnableWecom  bool   `yaml:"enable_wecom" json:"enable_wecom"`
        TelegramBotToken string `yaml:"telegram_bot_token" json:"telegram_bot_token"`
        TelegramChatID   string `yaml:"telegram_chat_id" json:"telegram_chat_id"`
        EnableTelegram   bool   `yaml:"enable_telegram" json:"enable_telegram"`
    } `yaml:"notify" json:"notify"`
}

type Event struct {
    SrcIP uint32
}

var cfg Config
var lastIP string
var lastUpdate time.Time
var updatePeriod time.Duration
var updatesTotal int64
var updateErrorsTotal int64
var startTime time.Time

var hitMap = make(map[string][]time.Time)

const logFile = "/var/log/icmp-ddns.log"
