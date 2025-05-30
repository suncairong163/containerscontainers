package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// UserInfo 包含系统用户详细信息
type UserInfo struct {
	Username string
	UID      int
	GID      int
	Name     string
	HomeDir  string
	Shell    string
}

func main() {
	fmt.Println("系统所有账号信息:")
	fmt.Println("==================")

	// 获取所有用户信息
	users, err := getAllUsers()
	if err != nil {
		fmt.Printf("获取用户信息失败: %v\n", err)
		return
	}

	// 打印用户信息表格
	printUserTable(users)

	// 打印组信息
	printGroupInfo()
}

// getAllUsers 获取系统所有用户信息
func getAllUsers() ([]UserInfo, error) {
	switch runtime.GOOS {
	case "linux", "darwin":
		return getUnixUsers()
	case "windows":
		return getWindowsUsers()
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// getUnixUsers 获取类Unix系统用户信息
func getUnixUsers() ([]UserInfo, error) {
	var users []UserInfo

	// 方法1: 使用os/user包（只获取当前用户，无法获取所有用户）
	// 此方法有局限性，但保留作为参考

	// 方法2: 读取/etc/passwd文件
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, fmt.Errorf("无法打开/etc/passwd文件: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue // 跳过注释行
		}

		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue // 跳过格式不正确的行
		}

		uid, _ := strconv.Atoi(parts[2])
		gid, _ := strconv.Atoi(parts[3])

		users = append(users, UserInfo{
			Username: parts[0],
			UID:      uid,
			GID:      gid,
			Name:     parts[4],
			HomeDir:  parts[5],
			Shell:    parts[6],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取/etc/passwd文件出错: %v", err)
	}

	// 按UID排序
	sort.Slice(users, func(i, j int) bool {
		return users[i].UID < users[j].UID
	})

	return users, nil
}

// getWindowsUsers 获取Windows系统用户信息
func getWindowsUsers() ([]UserInfo, error) {
	var users []UserInfo

	// 方法1: 使用net user命令
	cmd := exec.Command("net", "user")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行net user命令失败: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	// 跳过标题行
	for i := 0; i < 4 && scanner.Scan(); i++ {
		// 跳过前4行
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.Contains(line, "命令成功完成") {
			continue
		}

		usernames := strings.Fields(line)
		for _, username := range usernames {
			// 跳过系统保留账户
			if isReservedWindowsAccount(username) {
				continue
			}

			// 获取用户详细信息
			u, err := getUserDetails(username)
			if err == nil {
				users = append(users, u)
			}
		}
	}

	// 方法2: 使用WMI查询（更全面）
	wmiUsers, err := getWindowsUsersViaWMI()
	if err == nil && len(wmiUsers) > 0 {
		users = wmiUsers
	}

	return users, nil
}

// getUserDetails 获取Windows用户详细信息
func getUserDetails(username string) (UserInfo, error) {
	// 使用net user命令获取用户详情
	cmd := exec.Command("net", "user", username)
	output, err := cmd.Output()
	if err != nil {
		return UserInfo{}, err
	}

	userInfo := UserInfo{Username: username}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "全名") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) > 1 {
				userInfo.Name = strings.TrimSpace(parts[1])
			}
		}
	}

	return userInfo, nil
}

// getWindowsUsersViaWMI 使用WMI获取Windows用户信息
func getWindowsUsersViaWMI() ([]UserInfo, error) {
	cmd := exec.Command("wmic", "useraccount", "get", "name,sid")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行wmic命令失败: %v", err)
	}

	var users []UserInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// 跳过标题行
	if scanner.Scan() {
		// 跳过标题
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		username := parts[0]
		sid := parts[1]

		// 从SID中提取UID（Windows的UID等价物）
		uid := 0
		sidParts := strings.Split(sid, "-")
		if len(sidParts) > 1 {
			// 使用SID的最后一部分作为伪UID
			if id, err := strconv.Atoi(sidParts[len(sidParts)-1]); err == nil {
				uid = id
			}
		}

		// 跳过系统保留账户
		if isReservedWindowsAccount(username) {
			continue
		}

		users = append(users, UserInfo{
			Username: username,
			UID:      uid,
			Name:     username,
		})
	}

	return users, nil
}

// isReservedWindowsAccount 检查是否为Windows保留账户
func isReservedWindowsAccount(username string) bool {
	reservedAccounts := []string{
		"Administrator", "Guest", "DefaultAccount",
		"WDAGUtilityAccount", "system", "LOCAL SERVICE",
		"NETWORK SERVICE",
	}

	for _, acc := range reservedAccounts {
		if strings.EqualFold(username, acc) {
			return true
		}
	}
	return false
}

// printUserTable 打印用户信息表格
func printUserTable(users []UserInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// 打印表头
	if runtime.GOOS == "windows" {
		fmt.Fprintln(w, "用户名\tUID\t全名")
	} else {
		fmt.Fprintln(w, "用户名\tUID\tGID\t全名\t主目录\tShell")
	}

	// 打印用户信息
	for _, user := range users {
		if runtime.GOOS == "windows" {
			fmt.Fprintf(w, "%s\t%d\t%s\n",
				user.Username, user.UID, user.Name)
		} else {
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\n",
				user.Username, user.UID, user.GID, user.Name, user.HomeDir, user.Shell)
		}
	}

	w.Flush()
	fmt.Println()
}

// printGroupInfo 打印组信息（仅Unix系统）
func printGroupInfo() {
	if runtime.GOOS == "windows" {
		return
	}

	fmt.Println("\n系统组信息:")
	fmt.Println("=============")

	// 读取/etc/group文件
	file, err := os.Open("/etc/group")
	if err != nil {
		fmt.Printf("无法打开/etc/group文件: %v\n", err)
		return
	}
	defer file.Close()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "组名\tGID\t成员")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue // 跳过注释行
		}

		parts := strings.Split(line, ":")
		if len(parts) < 4 {
			continue
		}

		group := parts[0]
		gid := parts[2]
		members := parts[3]

		fmt.Fprintf(w, "%s\t%s\t%s\n", group, gid, members)
	}

	w.Flush()
}

// getCurrentUserInfo 获取当前用户信息（示例函数）
func getCurrentUserInfo() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("获取当前用户信息失败: %v\n", err)
		return
	}

	fmt.Printf("\n当前用户信息:\n")
	fmt.Printf("用户名: %s\n", currentUser.Username)
	fmt.Printf("UID: %s\n", currentUser.Uid)
	fmt.Printf("GID: %s\n", currentUser.Gid)
	fmt.Printf("全名: %s\n", currentUser.Name)
	fmt.Printf("主目录: %s\n", currentUser.HomeDir)

	// 获取登录信息（仅Unix）
	if runtime.GOOS != "windows" {
		if sysUser, err := user.LookupId(currentUser.Uid); err == nil {
			fmt.Printf("Shell: %s\n", sysUser.Username)
		}
	}
}
