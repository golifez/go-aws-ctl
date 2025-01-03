package lightsail

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/golifez/go-aws-ctl/cmd"
	cmd2 "github.com/golifez/go-aws-ctl/cmd"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	"github.com/aws/aws-sdk-go-v2/service/lightsail/types"
	"github.com/spf13/cobra"
)

// FirewalldCmd represents the create command
// # 打开所有端口
// go run main.go lg  Fw --region ap-northeast-2  --instanceNames all --ports all
//
// # 打开具体实例的 80 和 443 端口
// go run main.go lg  Fw --regionap-northeast-2 --instanceNames instance1 --ports 80,443
//
// # 打开多个实例的 80-100 端口
// go run main.go lg  Fw --region ap-northeast-2   --instanceNames instance1,instance2 --ports 80-100
var FirewalldCmd = &cobra.Command{
	Use:   "Fw",
	Short: "Enable firewall for Lightsail instances",
	Long:  `Enable specific ports or a range of ports for AWS Lightsail instances.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取命令行参数
		region, _ := cmd.Flags().GetString("region")
		instanceNames, _ := cmd.Flags().GetStringSlice("instanceNames")
		ports, _ := cmd.Flags().GetStringSlice("ports")

		// 获取 Lightsail 客户端
		cl := cmd2.GetClient[*lightsail.Client](cmd2.WithRegion(region), cmd2.WithClientType("lightsail"))

		// 开启防火墙端口
		if err := openPorts(cl, instanceNames, ports); err != nil {
			fmt.Printf("Failed to open ports: %v\n", err)
		} else {
			fmt.Println("Ports opened successfully.")
		}
	},
}

var ctx = context.Background()

func init() {
	cmd.LgCmd.AddCommand(FirewalldCmd)
	FirewalldCmd.Flags().StringP("region", "r", "", "Region (e.g., us-east-1)")
	FirewalldCmd.Flags().StringSlice("instanceNames", []string{"all"}, "Instance names (e.g., name1,name2)")
	FirewalldCmd.Flags().StringSlice("ports", []string{"all"}, "Ports to open (e.g., 80,443 or 0-65535)")
}

// openPorts opens the specified ports for Lightsail instances.
func openPorts(client *lightsail.Client, instanceNames, ports []string) error {
	var instances []string

	// 处理实例名称
	if len(instanceNames) == 1 && instanceNames[0] == "all" {
		output, err := client.GetInstances(ctx, &lightsail.GetInstancesInput{})
		if err != nil {
			return fmt.Errorf("failed to get instances: %w", err)
		}
		for _, instance := range output.Instances {
			instances = append(instances, aws.ToString(instance.Name))
		}
	} else {
		instances = instanceNames
	}

	// 遍历每个实例并打开指定的端口
	for _, instance := range instances {
		for _, port := range ports {
			if port == "all" {
				// 打开所有端口 (0-65535) 和所有协议
				_, err := client.OpenInstancePublicPorts(ctx, &lightsail.OpenInstancePublicPortsInput{
					InstanceName: aws.String(instance),
					PortInfo: &types.PortInfo{
						FromPort: 0,
						ToPort:   65535,
						Protocol: types.NetworkProtocolAll,
					},
				})
				if err != nil {
					return fmt.Errorf("failed to open all ports for instance %s: %w", instance, err)
				}
			} else {
				// 打开具体的端口
				portRange, err := parsePortRange(port)
				if err != nil {
					return fmt.Errorf("invalid port %s for instance %s: %w", port, instance, err)
				}

				_, err = client.OpenInstancePublicPorts(ctx, &lightsail.OpenInstancePublicPortsInput{
					InstanceName: aws.String(instance),
					PortInfo: &types.PortInfo{
						FromPort: portRange[0],
						ToPort:   portRange[1],
						Protocol: types.NetworkProtocolTcp,
					},
				})
				if err != nil {
					return fmt.Errorf("failed to open port %s for instance %s: %w", port, instance, err)
				}
			}
		}
	}
	return nil
}

// 解析端口范围

// 传入 80-100 或者 80 返回 [80,100] 或者 [80,80]
func parsePortRange(port string) ([]int32, error) {
	var ports []int32
	// 80-100
	if strings.Contains(port, "-") {
		parts := strings.Split(port, "-")
		from, _ := strconv.Atoi(parts[0]) // 把字符串转换为 int
		to, _ := strconv.Atoi(parts[1])
		return []int32{int32(from), int32(to)}, nil
	}
	// 80,443
	if strings.Contains(port, ",") {
		parts := strings.Split(port, ",")
		for _, p := range parts {
			val, _ := strconv.Atoi(p)
			ports = append(ports, int32(val))
		}
		return ports, nil
	}
	// 80
	val, _ := strconv.Atoi(port)
	return []int32{int32(val)}, nil
}

// 打开防火墙端口
func openFirewallPort(client *lightsail.Client, instanceName string, ports string) error {
	// 解析端口
	portRange, err := parsePortRange(ports)
	if err != nil {
		return fmt.Errorf("invalid port %s for instance %s: %w", ports, instanceName, err)
	}
	// 打开端口
	protocol := types.NetworkProtocolTcp
	// 如果是全部端口范围，则使用所有协议
	if portRange[0] == 0 && portRange[1] == 65535 {
		protocol = types.NetworkProtocolAll
	}

	_, err = client.OpenInstancePublicPorts(ctx, &lightsail.OpenInstancePublicPortsInput{
		InstanceName: aws.String(instanceName),
		PortInfo: &types.PortInfo{
			FromPort: portRange[0],
			ToPort:   portRange[1],
			Protocol: protocol,
		},
	})
	return err
}

// 查询实例防火墙端口
func QueryInstanceFirewallPort(client *lightsail.Client, instanceName string) error {
	// 查询端口
	out, err := client.GetInstancePortStates(ctx, &lightsail.GetInstancePortStatesInput{
		InstanceName: aws.String(instanceName),
	})
	fmt.Println(out.PortStates)
	return err
}
