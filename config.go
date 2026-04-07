package main

import (
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

var configPath = "./config.json"

func loadConfig() {
    data, err := ioutil.ReadFile(configPath)
    if err != nil {
        log.Fatal(err)
    }
    if err := json.Unmarshal(data, &cfg); err != nil {
        log.Fatal(err)
    }
    updatePeriod = time.Duration(cfg.Runtime.UpdateInterval) * time.Second
    if cfg.Runtime.AdminAddr == "" {
        cfg.Runtime.AdminAddr = ":8080"
    }
    if cfg.Runtime.AdminRateLimitPerMin == 0 {
        cfg.Runtime.AdminRateLimitPerMin = 60
    }
}

func copyFile(src, dst string) error {
    in, err := os.Open(src)
    if err != nil {
        return err
    }
    defer in.Close()

    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer out.Close()

    if _, err := io.Copy(out, in); err != nil {
        return err
    }
    return out.Sync()
}

// saveConfig saves current in-memory cfg to disk atomically and returns the
// backup path if an existing config was backed up (empty if none).
func saveConfig() (string, error) {
    dir := filepath.Dir(configPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return "", err
    }

    backupsDir := filepath.Join(dir, "backups")

    // If existing config exists, create backup
    if _, err := os.Stat(configPath); err == nil {
        if err := os.MkdirAll(backupsDir, 0755); err != nil {
            return "", err
        }
        ts := time.Now().Format("20060102-150405")
        backup := filepath.Join(backupsDir, fmt.Sprintf("config-%s.yaml", ts))
        if err := copyFile(configPath, backup); err != nil {
            return "", err
        }

            // write new config atomically
            tmp := configPath + ".tmp"
            data, err := json.MarshalIndent(&cfg, "", "  ")
            if err != nil {
                return "", err
            }
            if err := ioutil.WriteFile(tmp, data, 0644); err != nil {
                return "", err
            }
            if err := os.Rename(tmp, configPath); err != nil {
                return "", err
            }
        return backup, nil
    } else if os.IsNotExist(err) {
        // No existing config, just write atomically
        tmp := configPath + ".tmp"
        data, err := json.MarshalIndent(&cfg, "", "  ")
        if err != nil {
            return "", err
        }
        if err := ioutil.WriteFile(tmp, data, 0644); err != nil {
            return "", err
        }
        if err := os.Rename(tmp, configPath); err != nil {
            return "", err
        }
        return "", nil
    } else {
        return "", err
    }
}

func getIfIndex(name string) int {
    out, _ := exec.Command("cat", "/sys/class/net/"+name+"/ifindex").Output()
    var idx int
    fmt.Sscanf(string(out), "%d", &idx)
    return idx
}
