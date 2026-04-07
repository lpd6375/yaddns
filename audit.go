package main

import (
    "encoding/json"
    "log"
    "os"
    "time"
)

var auditLogPath = "/var/log/icmp-ddns-admin.log"

// auditEntry is a single-line JSON audit record
type auditEntry struct {
    Ts      string `json:"ts"`
    User    string `json:"user"`
    IP      string `json:"ip"`
    Action  string `json:"action"`
    Details string `json:"details,omitempty"`
}

func writeAudit(user, action, ip, details string) {
    e := auditEntry{
        Ts:      time.Now().UTC().Format(time.RFC3339),
        User:    user,
        IP:      ip,
        Action:  action,
        Details: details,
    }
    b, err := json.Marshal(e)
    if err != nil {
        log.Println("audit marshal failed:", err)
        return
    }

    // Ensure directory exists
    if f, err := os.OpenFile(auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
        defer f.Close()
        f.Write(append(b, '\n'))
    } else {
        log.Println("unable to write audit log:", err)
    }
}
