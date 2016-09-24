/*
 * Project: awsgo
 * File: swapip.go
 *
 * Copyright (c) 2016 Sanjeewa Wijesundara
 *
 */

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {

	primaryAllocation := os.Getenv("AWS_PRIMARY_IP_ALLOCAION")
	secondaryAllocation := os.Getenv("AWS_SECONDARY_IP_ALLOCATION")

	instanceAZA := os.Getenv("AWS_INSTANCE_AZA")
	instanceAZB := os.Getenv("AWS_INSTANCE_AZB")

	fmt.Println("Checking configuration")

	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		fmt.Println(pair[0])
	}

	// Create an EC2 service object in the "ap-southeast-2" region
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String("ap-southeast-2")})

	IGInstances := []*string{
		aws.String(instanceAZA), // AZA
		aws.String(instanceAZB), // AZB
	}

	// Call the DescribeInstances Operation
	param := &ec2.DescribeInstancesInput{
		DryRun:      aws.Bool(false),
		InstanceIds: IGInstances,
	}

	resp, err := svc.DescribeInstances(param)
	if err != nil {
		// Print the error, cast err to aws.Error to get the Code and message.
		panic(err.Error())
	}

	for idx := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			InstState := *inst.State
			switch *InstState.Name {
			case ec2.InstanceStateNameRunning:
				fmt.Println("Instance", *inst.InstanceId, "state is running")
			case ec2.InstanceStateNameStopped:
				fmt.Println("Instance", *inst.InstanceId, "state is stopped")
			default:
				fmt.Println("Instance state is ", *InstState.Name)
			}

		}
	}

	dissassociateEIP(instanceAZA, svc)
	dissassociateEIP(instanceAZB, svc)

	associateEIP(instanceAZA, secondaryAllocation, svc)
	associateEIP(instanceAZB, primaryAllocation, svc)
}

// function to associate an ip allocation to an instance
func associateEIP(instanceID string, allocationID string, svc *ec2.EC2) {
	fmt.Println("Trying to assign ", allocationID, "to ", instanceID)

	// Setup parameters for allocation
	param := &ec2.AssociateAddressInput{
		DryRun:             aws.Bool(false),
		InstanceId:         aws.String(instanceID),
		AllocationId:       aws.String(allocationID),
		AllowReassociation: aws.Bool(true),
	}
	resp, err := svc.AssociateAddress(param)

	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Association successful", *resp.AssociationId)
}

// function to get IP association ID from an instance
func getAssociationID(instanceID string, svc *ec2.EC2) (string, bool) {
	fmt.Println("Trying to find association id of", instanceID)

	param := &ec2.DescribeAddressesInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(instanceID), // Required
				},
			},
		},
	}
	resp, err := svc.DescribeAddresses(param)

	if err != nil {
		panic(err.Error())
	}

	if len(resp.Addresses) != 1 {
		return "Association not found", false
	}

	return *resp.Addresses[0].AssociationId, true

}

// function to dissassociate an association
func dissassociateEIP(instanceID string, svc *ec2.EC2) {
	fmt.Println("Trying to dissassociate ip on", instanceID)

	// Get current association id
	associationID, status := getAssociationID(instanceID, svc)

	if status == false {
		fmt.Println("Association not available")
		return
	}

	fmt.Println("Association found", associationID, ", now trying to remove it")

	param := &ec2.DisassociateAddressInput{
		DryRun:        aws.Bool(false),
		AssociationId: aws.String(associationID),
	}
	_, err := svc.DisassociateAddress(param)

	if err != nil {
		panic(err.Error())
	}
}
