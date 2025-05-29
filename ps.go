package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

var blackList = []string{"bash -c export LANG=", "sleep"}

func main() {
	// 获取当前所有进程
	initialProcesses, err := process.Processes()
	if err != nil {
		log.Fatalf("获取进程列表失败: %v", err)
	}

	// 创建进程状态跟踪器
	tracker := NewProcessTracker()
	for _, p := range initialProcesses {
		tracker.Add(p.Pid)
	}

	// 设置信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	fmt.Println("开始监听进程事件 (Ctrl+C 退出)...")

	// 事件处理循环
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("\n停止监听...")
			return

		case <-ticker.C:
			// 获取当前进程列表
			currentProcesses, err := process.Processes()
			if err != nil {
				log.Printf("获取进程列表失败: %v", err)
				continue
			}

			// 检测新进程
			currentPIDs := make(map[int32]bool)
			for _, p := range currentProcesses {
				pid := p.Pid
				currentPIDs[pid] = true
				//
				//if !tracker.Exists(pid) {
				//	tracker.Add(pid)
				//fmt.Printf("[进程启动] PID: %d\n", pid)

				// 获取进程详情
				if proc, err := process.NewProcess(pid); err == nil {

					tag := false
					for _, s := range blackList {

						if cmdline, err := proc.Cmdline(); err == nil {

							if strings.Contains(cmdline, s) {
								tag = true
							}
						}
					}

					if tag {
						continue
					}

					if name, err := proc.Name(); err == nil {
						fmt.Printf("  名称: %s\n", name)
					}
					if cmdline, err := proc.Cmdline(); err == nil {
						fmt.Printf("  命令行: %s\n", cmdline)
					}

					fmt.Printf("  命令行: %s\n", proc)
				}
				//}
				//}
				//
				//// 检测已退出的进程
				//for pid := range tracker.processes {
				//	if !currentPIDs[pid] {
				//		tracker.Remove(pid)
				//
				//		//fmt.Printf("[进程退出] PID: %d\n", pid)
				//	}
			}
		}
	}
}

// ProcessTracker 跟踪进程状态
type ProcessTracker struct {
	processes map[int32]bool
}

func NewProcessTracker() *ProcessTracker {
	return &ProcessTracker{
		processes: make(map[int32]bool),
	}
}

func (t *ProcessTracker) Add(pid int32) {
	t.processes[pid] = true
}

func (t *ProcessTracker) Remove(pid int32) {
	delete(t.processes, pid)
}

func (t *ProcessTracker) Exists(pid int32) bool {
	_, exists := t.processes[pid]
	return exists
}
