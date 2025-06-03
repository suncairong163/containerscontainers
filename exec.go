package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("使用方法: sudo ./ns-exec <PID> <命令> [参数...]")
		fmt.Println("示例: sudo ./ns-exec 1234 /bin/ls -l /")
		fmt.Println("示例: sudo ./ns-exec 1234 /bin/sh -c \"ls -l /\"")
		os.Exit(1)
	}

	pid, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("无效的PID: %v", err)
	}

	command := os.Args[2:]
	if len(command) == 0 {
		log.Fatal("未指定要执行的命令")
	}

	fmt.Printf("准备进入PID %d 的命名空间执行命令: %v\n", pid, command)

	// 使用绝对路径执行命令
	absCommand, err := resolveCommandPath(command)
	if err != nil {
		log.Fatalf("无法解析命令路径: %v", err)
	}
	fmt.Printf("使用绝对路径执行: %v\n", absCommand)

	// 构建nsenter命令
	nsenterCmd := exec.Command(
		"nsenter",
		"-t", strconv.Itoa(pid), // 目标PID
		"-m", // 挂载命名空间 (文件系统视图)
		"-u", // UTS命名空间 (主机名和域名)
		"-i", // IPC命名空间 (进程间通信)
		"-n", // 网络命名空间 (网络接口)
		"-p", // PID命名空间 (进程树)
		"--",
	)
	nsenterCmd.Args = append(nsenterCmd.Args, absCommand...)

	nsenterCmd.Stdin = os.Stdin
	nsenterCmd.Stdout = os.Stdout
	nsenterCmd.Stderr = os.Stderr

	fmt.Println("执行结果:")
	fmt.Println("====================================")
	if err := nsenterCmd.Run(); err != nil {
		log.Fatalf("命令执行失败: %v", err)
	}
	fmt.Println("====================================")
	fmt.Println("命令执行完成")
}

// 尝试解析命令的绝对路径
func resolveCommandPath(command []string) ([]string, error) {
	// 如果已经是绝对路径，直接返回
	if strings.HasPrefix(command[0], "/") {
		return command, nil
	}

	// 尝试查找命令路径
	path, err := exec.LookPath(command[0])
	if err != nil {
		// 如果找不到，尝试常见路径
		commonPaths := []string{
			"/bin/" + command[0],
			"/usr/bin/" + command[0],
			"/sbin/" + command[0],
			"/usr/sbin/" + command[0],
		}

		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				newCommand := []string{p}
				newCommand = append(newCommand, command[1:]...)
				return newCommand, nil
			}
		}

		return nil, fmt.Errorf("无法找到命令 '%s'，尝试使用绝对路径", command[0])
	}

	newCommand := []string{path}
	newCommand = append(newCommand, command[1:]...)
	return newCommand, nil
}
