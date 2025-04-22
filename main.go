/*
 * @Descripttion: 微服务Demo
 * @version: 1.0
 * @Author: hujianghong
 * @Date: 2025-04-22 21:47:28
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-22 22:01:03
 */
package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("请指定要运行的服务: api-gateway 或 user-service")
		os.Exit(1)
	}

	serviceName := os.Args[1]

	switch serviceName {
	case "api-gateway":
		runAPIGateway()
	case "user-service":
		runUserService()
	default:
		fmt.Printf("未知的服务: %s\n", serviceName)
		os.Exit(1)
	}
}

func runAPIGateway() {
	cmd := exec.Command("go", "run", "./cmd/api-gateway/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("正在启动 API 网关服务...")
	if err := cmd.Run(); err != nil {
		fmt.Printf("启动 API 网关服务失败: %v\n", err)
		os.Exit(1)
	}
}

func runUserService() {
	cmd := exec.Command("go", "run", "./cmd/user-service/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("正在启动用户服务...")
	if err := cmd.Run(); err != nil {
		fmt.Printf("启动用户服务失败: %v\n", err)
		os.Exit(1)
	}
}
