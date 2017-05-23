package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	region  string
	down    string
	asgName string
)

func main() {
	flag.StringVar(&region, "region", "us-west-1", "AWS region")
	flag.StringVar(&asgName, "autoscaling-group-name", "", "autoscaling group to target for scale down")
	flag.StringVar(&down, "down", "", "specific IP to scale down within autoscaling group")
	flag.Parse()

	if asgName == "" {
		fmt.Println("must pass autoscaling group name to -autoscaling-group-name")
		os.Exit(-1)
	}

	if down == "" {
		fmt.Println("must pass IP address to -down")
		os.Exit(-1)
	}

	scaleDown()
}

func scaleDown() {
	protectedHosts := map[string]string{}
	unprotectedHosts := map[string]string{}
	asg := autoscaling.New(session.New(), &aws.Config{Region: aws.String(region)})
	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(asgName),
		},
	}

	resp, err := asg.DescribeAutoScalingGroups(params)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	svc := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})

	for _, g := range resp.AutoScalingGroups {
		for _, i := range g.Instances {
			params := &ec2.DescribeInstancesInput{
				InstanceIds: []*string{
					aws.String(*i.InstanceId),
				},
			}
			resp, err := svc.DescribeInstances(params)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
			for _, r := range resp.Reservations {
				for _, i := range r.Instances {
					if len(i.NetworkInterfaces) > 0 {
						ip := string(*(i.NetworkInterfaces[0].PrivateIpAddress))
						if ip != down {
							protectedHosts[string(*i.InstanceId)] = ip
							continue
						}
						unprotectedHosts[string(*i.InstanceId)] = ip
					}
				}
			}
		}
	}

	p := &autoscaling.SetInstanceProtectionInput{
		AutoScalingGroupName: aws.String(asgName),
		InstanceIds:          []*string{},
		ProtectedFromScaleIn: aws.Bool(true),
	}
	for k := range protectedHosts {
		p.InstanceIds = append(p.InstanceIds, aws.String(k))
	}
	fmt.Printf("%d hosts running\n", len(protectedHosts)+len(unprotectedHosts))
	fmt.Printf("setting scale protection true for %d hosts\n", len(protectedHosts))
	_, err = asg.SetInstanceProtection(p)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	p.InstanceIds = []*string{}
	p.ProtectedFromScaleIn = aws.Bool(false)
	for k := range unprotectedHosts {
		p.InstanceIds = append(p.InstanceIds, aws.String(k))
	}
	fmt.Printf("setting scale protection false for %d hosts\n", len(unprotectedHosts))
	_, err = asg.SetInstanceProtection(p)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	size := int64(len(protectedHosts))
	downParams := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgName),
		DesiredCapacity:      aws.Int64(size),
		MinSize:              aws.Int64(size),
	}
	fmt.Printf("scaling down cluster to %d instances\n", size)
	_, err = asg.UpdateAutoScalingGroup(downParams)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
