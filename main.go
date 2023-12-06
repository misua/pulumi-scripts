package main

import (
	"fmt"
	"os/exec"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/elb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Check if AWS SDK is installed
		_, err := exec.LookPath("aws")
		if err != nil {
			fmt.Println("AWS SDK is not installed. Installing...")
			cmd := exec.Command("bash", "-c", `
			curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
			unzip awscliv2.zip
			sudo ./aws/install
			`)
			err = cmd.Run()
			if err != nil {
				fmt.Println("Failed to install AWS SDK:", err)
				return err
			}
		}

		// Check if Pulumi is installed
		_, err = exec.LookPath("pulumi")
		if err != nil {
			fmt.Println("Pulumi is not installed. Installing...")
			cmd := exec.Command("bash", "-c", `
			curl -fsSL https://get.pulumi.com | sh
			`)
			err = cmd.Run()
			if err != nil {
				fmt.Println("Failed to install Pulumi:", err)
				return err
			}
		}

		fmt.Println("AWS SDK and Pulumi are installed.")

		// Your Pulumi program here...
		availabilityZones := []string{"us-west-1a", "us-west-1b", "us-west-1c"}
		var instances []*ec2.Instance
		var instanceIds []pulumi.StringInput

		for i, az := range availabilityZones {
			server, err := ec2.NewInstance(ctx,
				"web-server-"+string(i),
				&ec2.InstanceArgs{
					Ami:              pulumi.String("ami-0c94855ba95c574c8"), // Ubuntu Server 20.04 LTS
					InstanceType:     pulumi.String("t2.micro"),
					AvailabilityZone: pulumi.String(az),
					Tags:             pulumi.StringMap{"Name": pulumi.String("web-server-" + string(i))},
					KeyName:          pulumi.String("<your-key-name>"),
				})
			if err != nil {
				return err
			}

			instances = append(instances, server)
			instanceIds = append(instanceIds, server.ID())
		}

		group, err := ec2.NewSecurityGroup(ctx,
			"web-secgrp",
			&ec2.SecurityGroupArgs{
				Ingress: ec2.SecurityGroupIngressArray{
					&ec2.SecurityGroupIngressArgs{
						Protocol:   pulumi.String("tcp"),
						FromPort:   pulumi.Int(80),
						ToPort:     pulumi.Int(80),
						CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
					},
				},
				Egress: ec2.SecurityGroupEgressArray{
					&ec2.SecurityGroupEgressArgs{
						Protocol:   pulumi.String("-1"),
						FromPort:   pulumi.Int(0),
						ToPort:     pulumi.Int(0),
						CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
					},
				},
			})
		if err != nil {
			return err
		}

		lb, err := elb.NewLoadBalancer(ctx,
			"web-lb",
			&elb.LoadBalancerArgs{
				AvailabilityZones: pulumi.StringArray{
					pulumi.String("us-west-1a"),
					pulumi.String("us-west-1b"),
					pulumi.String("us-west-1c"),
				},
				Listeners: elb.LoadBalancerListenerArray{
					&elb.LoadBalancerListenerArgs{
						InstancePort:     pulumi.Int(80),
						InstanceProtocol: pulumi.String("http"),
						LbPort:           pulumi.Int(80),
						LbProtocol:       pulumi.String("http"),
					},
				},
				SecurityGroups: pulumi.StringArray{group.ID()},
			})
		if err != nil {
			return err
		}

		_, err = elb.NewAttachment(ctx,
			"web-lb-attachment",
			&elb.AttachmentArgs{
				Elb:      lb.ID(),
				Instance: pulumi.StringArray(instanceIds),
			})
		if err != nil {
			return err
		}

		return nil
	})
}
