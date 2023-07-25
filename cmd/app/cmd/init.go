/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/iDukeLu/kubengr/scripts"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCmd := exec.Command("sh", "-c", scripts.Script, "--", "master", hostName)

		outputPipe, err := initCmd.StdoutPipe()
		if err != nil {
			fmt.Printf("无法创建输出管道：%v\n", err)
			return
		}

		if err := initCmd.Start(); err != nil {
			fmt.Printf("无法启动命令：%v\n", err)
			return
		}

		buffer := make([]byte, 10240)
		for {
			n, err := outputPipe.Read(buffer)
			if err != nil {
				break
			}

			output := string(buffer[:n])
			output = strings.TrimSpace(output)
			if output != "" {
				fmt.Println(output)
			}
		}

		if err := initCmd.Wait(); err != nil {
			fmt.Printf("命令执行出错：%v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	initCmd.Flags().StringVarP(&hostName, "host-name", "n", "", "host name, eg: kube-mater、kube-worker1..")
	initCmd.MarkFlagRequired("host-name")
}
