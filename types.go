package main

import "time"

type Config struct {
    Interface string `yaml:"interface"`
    Cloudflare struct {
        Token    string `yaml:"token"`
        ZoneID   string `yaml:"zone_id"`
        RecordID string `yaml:"record_id"`
        Name     string `yaml:"name"`
        TTL      int    `yaml:"ttl"`
    } `yaml:"cloudflare"`
    Runtime struct {
        UpdateInterval int `yaml:"update_interval"`
        AdminAddr      string `yaml:"admin_addr"`
        AdminToken     string `yaml:"admin_token"`
        AdminIPWhitelist []string `yaml:"admin_ip_whitelist"`
        AdminRateLimitPerMin int `yaml:"admin_rate_limit_per_min"`
    } `yaml:"runtime"`
    Notify struct {
        WecomWebhook string `yaml:"wecom_webhook"`
    } `yaml:"notify"`
}

type Event struct {
    SrcIP uint32
}

var cfg Config
var lastIP string
var lastUpdate time.Time
var updatePeriod time.Duration

var hitMap = make(map[string][]time.Time)

const logFile = "/var/log/icmp-ddns.log"
