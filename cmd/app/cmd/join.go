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

// joinCmd represents the init command
var joinCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		joinCmd := exec.Command("sh", "-c", scripts.Script, "--", "worker", hostName, masterAddress, token, discoveryTokenCACertHash)

		outputPipe, err := joinCmd.StdoutPipe()
		if err != nil {
			fmt.Printf("无法创建输出管道：%v\n", err)
			return
		}

		if err := joinCmd.Start(); err != nil {
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

		if err := joinCmd.Wait(); err != nil {
			fmt.Printf("命令执行出错：%v\n", err)
			return
		}
	},
}

var masterAddress string
var token string
var discoveryTokenCACertHash string

func init() {
	rootCmd.AddCommand(joinCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// joinCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	joinCmd.Flags().StringVarP(&hostName, "host-name", "", "", "host name, eg: kube-mater、kube-worker1..")
	joinCmd.MarkFlagRequired("host-name")

	joinCmd.Flags().StringVarP(&masterAddress, "master-address", "", "", "master address, eg: 11.10.47.44:6443")
	joinCmd.MarkFlagRequired("master-address")

	joinCmd.Flags().StringVarP(&token, "token", "", "", "worker join token, get by command: kubeadm token create --print-join-command --ttl=0")
	joinCmd.MarkFlagRequired("token")

	joinCmd.Flags().StringVarP(&discoveryTokenCACertHash, "discovery-token-ca-cert-hash", "", "", "worker join token ca cert hash, get by command: kubeadm token create --print-join-command --ttl=0")
	joinCmd.MarkFlagRequired("discovery-token-ca-cert-hash")
}
