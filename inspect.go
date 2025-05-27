package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type FullInspectInfo struct {
	ContainerID string                 `json:"ContainerID"`
	Namespace   string                 `json:"Namespace"`
	Runtime     RuntimeInfo            `json:"Runtime"`
	Image       ImageInfo              `json:"Image"`
	Status      StatusInfo             `json:"Status"`
	Spec        specs.Spec             `json:"Spec"`
	Labels      map[string]string      `json:"Labels"`
	Annotations map[string]string      `json:"Annotations"`
	CreatedAt   string                 `json:"CreatedAt"`
	Snapshotter string                 `json:"Snapshotter"`
	Extensions  map[string]interface{} `json:"Extensions"`
	SnapshotKey string                 `json:"SnapshotKey"`
	Task        TaskInfo               `json:"Task"`
	NetworkSpec interface{}            `json:"NetworkSpec"`
	CgroupPath  string                 `json:"CgroupPath"`
	Checkpoint  interface{}            `json:"Checkpoint"`
}

type RuntimeInfo struct {
	Name    string                 `json:"Name"`
	Options map[string]interface{} `json:"Options"`
}

type ImageInfo struct {
	Name   string `json:"Name"`
	Digest string `json:"Digest"`
	Size   int64  `json:"Size"`
}

type StatusInfo struct {
	Status string `json:"Status"`
	Pid    int    `json:"Pid"`
}

type TaskInfo struct {
	Pid        int                    `json:"Pid"`
	Status     string                 `json:"Status"`
	IO         interface{}            `json:"IO"`
	Checkpoint interface{}            `json:"Checkpoint"`
	Options    map[string]interface{} `json:"Options"`
}

func main() {
	// 连接到containerd守护进程
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatalf("无法连接到containerd: %v", err)
	}
	defer client.Close()

	list, err := client.NamespaceService().List(context.Background())

	for _, s := range list {
		log.Println("namespace=====>>> %s ", s)
		ctx := namespaces.WithNamespace(context.Background(), s)

		containers, err := client.Containers(ctx)
		if err != nil {
			log.Fatalf("获取容器列表失败: %v", err)
		}

		for _, container := range containers {
			info, _ := container.Info(ctx)
			var spec specs.Spec
			json.Unmarshal(info.Spec.GetValue(), &spec)

			// 获取完整信息
			inspectInfo := FullInspectInfo{
				ContainerID: container.ID(),
				Namespace:   s,
				Runtime: RuntimeInfo{
					Name: info.Runtime.Name,
					//Options: info.Runtime.Options,
				},
				Labels:      info.Labels,
				Annotations: spec.Annotations,
				CreatedAt:   info.CreatedAt.String(),
				Snapshotter: info.Snapshotter,
				SnapshotKey: info.SnapshotKey,
				//Extensions:  info.Extensions,
				Spec:        spec,
				CgroupPath:  getCgroupPath(spec),
				NetworkSpec: getNetworkSpec(spec),
			}

			// 获取镜像信息
			if image, err := container.Image(ctx); err == nil {
				inspectInfo.Image = ImageInfo{
					Name:   image.Name(),
					Digest: image.Target().Digest.String(),
					Size:   image.Target().Size,
				}
			}

			// 获取任务信息
			if task, err := container.Task(ctx, nil); err == nil {
				status, _ := task.Status(ctx)
				inspectInfo.Status = StatusInfo{
					Status: string(status.Status),
					Pid:    int(task.Pid()),
				}
				inspectInfo.Task = TaskInfo{
					Pid:    int(task.Pid()),
					Status: string(status.Status),
					//Options: tas,
				}
			}

			// 转换为JSON格式输出
			jsonData, _ := json.MarshalIndent(inspectInfo, "", "  ")
			fmt.Printf("%s\n", jsonData)
			fmt.Println("----------------------------------------")
		}

	}
}

// 从OCI spec获取cgroup路径
func getCgroupPath(spec specs.Spec) string {
	if spec.Linux != nil {
		return spec.Linux.CgroupsPath
	}
	return ""
}

// 获取网络规范信息
func getNetworkSpec(spec specs.Spec) interface{} {
	if spec.Linux != nil {
		return map[string]interface{}{
			"namespaces":        spec.Linux.Namespaces,
			"resources":         spec.Linux.Resources,
			"sysctl":            spec.Linux.Sysctl,
			"devices":           spec.Linux.Devices,
			"seccomp":           spec.Linux.Seccomp,
			"rootfsPropagation": spec.Linux.RootfsPropagation,
		}
	}
	return nil
}
