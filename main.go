package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	cmd  = "/usr/bin/sudo"
	args = []string{"-u", "ec2-user", "ssh"}
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./essh <filter> [<filter>...] [ -- <command>]")
		os.Exit(1)
	}
	filters, command := ParseArgs(os.Args[1:])

	// Grab our instances
	instances := fetchInstances()
	// instances := []*ec2.Instance{
	// 	&ec2.Instance{
	// 		PrivateIpAddress: String("192.168.1.1"),
	// 		Tags: []*ec2.Tag{
	// 			&ec2.Tag{
	// 				Key:   String("Name"),
	// 				Value: String("my-instance"),
	// 			},
	// 		},
	// 	},
	// 	&ec2.Instance{
	// 		PrivateIpAddress: String("192.168.1.2"),
	// 		Tags: []*ec2.Tag{
	// 			&ec2.Tag{
	// 				Key:   String("Name"),
	// 				Value: String("my-app"),
	// 			},
	// 		},
	// 	},
	// }

	// Filter by argument (env, region, zone, app)
	filtered := filterInstances(filters, instances)

	if len(filtered) == 0 {
		fmt.Println("No match found")
		return
	}

	// One result and no args - exec ssh and give control back to the user
	if command == nil && len(filtered) == 1 {
		interactive(filtered[0])
		return
	}

	// If command is present, execute on all
	if len(command) > 0 {
		names := []string{}
		for _, i := range filtered {
			names = append(names, instanceTag(i, "Name"))
		}
		fmt.Printf("Matched nodes: %s\n", strings.Join(names, ", "))
		confirm()

		for _, i := range filtered {
			fmt.Printf("\n=== %s\n", instanceTag(i, "Name"))
			allArgs := append([]string{*i.PrivateIpAddress}, command...)
			cmd := exec.Command(cmd, append(args, allArgs...)...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("error executing command: %s\n", err.Error())
			}
		}
		return
	}

	// If command isn't present, request user filter down arguments
	i := giveChoice(filtered)
	interactive(i)
}

func fetchInstances() []*ec2.Instance {
	ec2svc := ec2.New(session.New())
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	}
	resp, err := ec2svc.DescribeInstances(params)
	if err != nil {
		fmt.Println("there was an error listing instances in", err.Error())
		os.Exit(1)
	}

	instances := []*ec2.Instance{}
	for idx := range resp.Reservations {
		instances = append(resp.Reservations[idx].Instances, instances...)
	}
	return instances
}

func filterInstances(filters []string, instances []*ec2.Instance) []*ec2.Instance {
	filtered := instances[:0]
	for _, i := range instances {
		if instanceHasFilter(i, filters) {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

func instanceHasFilter(i *ec2.Instance, filters []string) bool {
	for _, filter := range filters {
		if tagsHasFilter(i.Tags, filter) {
			return true
		}
	}
	return false
}

func tagsHasFilter(tags []*ec2.Tag, filter string) bool {
	for _, tag := range tags {
		if strings.Contains(tag.GoString(), filter) {
			return true
		}
	}
	return false
}

func instanceTag(i *ec2.Instance, key string) string {
	for _, tag := range i.Tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

func interactive(i *ec2.Instance) {
	allArgs := append([]string{cmd}, args...)
	syscall.Exec(cmd, append(allArgs, *i.PrivateIpAddress), []string{})
}

func giveChoice(filtered []*ec2.Instance) *ec2.Instance {
	fmt.Println("Multiple matches:")
	for idx, i := range filtered {
		fmt.Printf("[%d] %s %s\n", idx, instanceTag(i, "Name"), *i.PrivateIpAddress)
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Pick [0]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return filtered[0]
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 0 || idx > len(filtered)-1 {
		fmt.Println("Bad input")
		os.Exit(1)
	}
	return filtered[idx]
}

func confirm() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure you want to run this on these nodes? Ctrl+C to cancel?")
	reader.ReadString('\n')
}

func String(str string) *string {
	return &str
}

func ParseArgs(args []string) (filters []string, command []string) {
	if len(args) == 0 {
		return
	}

	isCommand := false
	for _, a := range args {
		if isCommand {
			command = append(command, a)
		} else if a == "--" {
			isCommand = true
		} else {
			filters = append(filters, a)
		}
	}

	return
}
