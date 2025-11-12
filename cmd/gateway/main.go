package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wsx864321/kim/internal/gateway/server"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "gateway",
	Short: "KIM Gateway Service",
	Long:  "KIM Gateway Service - 提供客户端连接网关服务",
	Run:   runGateway,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "配置文件路径 (required)")
	rootCmd.MarkFlagRequired("config")
}

func runGateway(cmd *cobra.Command, args []string) {
	if configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: 配置文件路径不能为空\n")
		cmd.Help()
		os.Exit(1)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: 配置文件不存在: %s\n", configPath)
		os.Exit(1)
	}

	// 启动服务
	server.Run(configPath)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

