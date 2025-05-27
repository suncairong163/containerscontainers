package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func main() {
	// 连接到containerd守护进程
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatalf("无法连接到containerd: %v", err)
	}
	defer client.Close()

	list, err := client.NamespaceService().List(context.Background())

	for _, s := range list {
		log.Println("当前namespace: %s",s)
		// 创建带有默认命名空间的上下文
		ctx := namespaces.WithNamespace(context.Background(), s)

		// 获取容器列表
		containers, err := client.Containers(ctx)
		if err != nil {
			log.Fatalf("获取容器列表失败: %v", err)
		}

		// 打印容器信息
		fmt.Println("Containerd容器列表:")
		for _, container := range containers {
			printContainerDetails(ctx, container)
			fmt.Println("----------------------------------------")
		}
	}




}

func printContainerDetails(ctx context.Context, container containerd.Container) {
	// 获取基础信息
	id := container.ID()
	fmt.Printf("容器ID: %s\n", id)

	// 获取容器完整信息
	info, err := container.Info(ctx)
	if err != nil {
		fmt.Printf("  → 错误: 无法获取容器信息 - %v\n", err)
		return
	}

	// 解析OCI配置
	var spec specs.Spec
	if err := json.Unmarshal(info.Spec.GetValue(), &spec); err != nil {
		fmt.Printf("  → 错误: 无法解析OCI配置 - %v\n", err)
	}

	// 打印基本信息
	fmt.Printf("创建时间: %s\n", info.CreatedAt.Format(time.RFC3339))
	fmt.Printf("运行时: %s (选项: %v)\n", info.Runtime.Name, info.Runtime.Options)
	fmt.Printf("标签: %v\n", info.Labels)

	// 获取镜像信息
	image, err := container.Image(ctx)
	if err == nil {
		fmt.Printf("镜像名称: %s\n", image.Name())
		fmt.Printf("镜像Digest: %s\n", image.Target().Digest)
	}

	// 打印状态信息
	task, err := container.Task(ctx, nil)
	if err == nil {
		status, _ := task.Status(ctx)
		fmt.Printf("运行状态: %s\n", status.Status)
		fmt.Printf("进程ID: %d\n", task.Pid())
	} else {
		fmt.Println("运行状态: 已停止")
	}

	// 打印OCI配置信息
	fmt.Println("\nOCI 运行时配置:")
	fmt.Printf("主机名: %s\n", spec.Hostname)
	fmt.Printf("工作目录: %s\n", spec.Process.Cwd)
	fmt.Printf("用户: UID=%d GID=%d\n", spec.Process.User.UID, spec.Process.User.GID)

	// 打印环境变量
	fmt.Println("\n环境变量:")
	for _, env := range spec.Process.Env {
		fmt.Printf("  - %s\n", env)
	}

	// 打印挂载点
	fmt.Println("\n挂载点:")
	for _, mount := range spec.Mounts {
		fmt.Printf("  - 类型: %-8s 源: %-20s 目标: %s\n",
			mount.Type, mount.Source, mount.Destination)
	}


	// 打印Annotations
	fmt.Println("\nAnnotations:")
	for k, v := range spec.Annotations {
		fmt.Printf("  %s: %s\n", k, v)
	}
}

