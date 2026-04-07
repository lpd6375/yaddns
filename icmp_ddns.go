package main

import (
	"log"
	"time"
)

func main() {
	startTime = time.Now()
	loadConfig()

	// 启动管理界面（后台运行）
	go startAdmin(cfg.Runtime.AdminAddr)

	if err := run(); err != nil {
		log.Fatal(err)
	}
}