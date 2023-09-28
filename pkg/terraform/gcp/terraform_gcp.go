// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package terraformGCP

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ava-labs/avalanche-cli/pkg/constants"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// SetCloudCredentials sets AWS account credentials defined in .aws dir in user home dir
func SetCloudCredentials(rootBody *hclwrite.Body, zone, credentialsPath, projectName string) error {
	// zone's format is us-east1-b, region's format is us-east1
	region := strings.Join(strings.Split(zone, "-")[:2], "-")
	provider := rootBody.AppendNewBlock("provider", []string{"google"})
	providerBody := provider.Body()
	providerBody.SetAttributeValue("project", cty.StringVal(projectName))
	providerBody.SetAttributeValue("region", cty.StringVal(region))
	providerBody.SetAttributeValue("zone", cty.StringVal(zone))
	providerBody.SetAttributeValue("credentials", cty.StringVal(credentialsPath))
	return nil
}

// SetNetwork houses the firewall (AWS security group equivalent) for GCP
func SetNetwork(rootBody *hclwrite.Body, ipAddress, networkName string) {
	network := rootBody.AppendNewBlock("resource", []string{"google_compute_network", networkName})
	networkBody := network.Body()
	networkBody.SetAttributeValue("name", cty.StringVal(networkName))
	SetFirewallRule(rootBody, "0.0.0.0/0", fmt.Sprintf("%s-%s", networkName, "default"), networkName, []string{strconv.Itoa(constants.AvalanchegoAPIPort), strconv.Itoa(constants.AvalanchegoP2PPort)})
	SetFirewallRule(rootBody, ipAddress+"/32", fmt.Sprintf("%s-%s", networkName, strings.ReplaceAll(ipAddress, ".", "")), networkName, []string{strconv.Itoa(constants.SSHTCPPort), strconv.Itoa(constants.AvalanchegoAPIPort)})
}

func SetFirewallRule(rootBody *hclwrite.Body, ipAddress, firewallName, networkName string, ports []string) {
	firewall := rootBody.AppendNewBlock("resource", []string{"google_compute_firewall", firewallName})
	firewallBody := firewall.Body()
	firewallBody.SetAttributeValue("name", cty.StringVal(firewallName))
	firewallBody.SetAttributeTraversal("network", hcl.Traversal{
		hcl.TraverseRoot{
			Name: "google_compute_network",
		},
		hcl.TraverseAttr{
			Name: networkName,
		},
		hcl.TraverseAttr{
			Name: "name",
		},
	})
	firewallAllow := firewallBody.AppendNewBlock("allow", []string{})
	firewallAllowBody := firewallAllow.Body()
	firewallAllowBody.SetAttributeValue("protocol", cty.StringVal("tcp"))
	var allowPortList []cty.Value
	for i := range ports {
		allowPortList = append(allowPortList, cty.StringVal(ports[i]))
	}
	firewallAllowBody.SetAttributeValue("ports", cty.ListVal(allowPortList))
	var allowIPList []cty.Value
	allowIPList = append(allowIPList, cty.StringVal(ipAddress))
	firewallBody.SetAttributeValue("source_ranges", cty.ListVal(allowIPList))
}

// SetPublicIP attach static IP(s) to the associated Google VM instance(s)
func SetPublicIP(rootBody *hclwrite.Body, nodeName string) {
	staticIPName := fmt.Sprintf("static-ip-%s", nodeName)
	eip := rootBody.AppendNewBlock("resource", []string{"google_compute_address", staticIPName})
	eipBody := eip.Body()
	eipBody.SetAttributeValue("name", cty.StringVal(staticIPName))
	eipBody.SetAttributeValue("address_type", cty.StringVal("EXTERNAL"))
	eipBody.SetAttributeValue("network_tier", cty.StringVal("PREMIUM"))
}

// SetupInstances adds aws_instance section in terraform state file where we configure all the necessary components of the desired ec2 instance(s)
func SetupInstances(rootBody *hclwrite.Body, networkName, sshPublicKey, ami, staticIPName, instanceName, keyPairName string) {
	gcpInstance := rootBody.AppendNewBlock("resource", []string{"google_compute_instance", "gcp-node"})
	gcpInstanceBody := gcpInstance.Body()
	gcpInstanceBody.SetAttributeValue("name", cty.StringVal(instanceName))
	gcpInstanceBody.SetAttributeValue("machine_type", cty.StringVal("e2-standard-8"))
	metadataMap := make(map[string]cty.Value)
	metadataMap["ssh-keys"] = cty.StringVal(fmt.Sprintf("%s:%s", keyPairName, strings.TrimSuffix(sshPublicKey, "\n")))
	gcpInstanceBody.SetAttributeValue("metadata", cty.ObjectVal(metadataMap))
	networkInterface := gcpInstanceBody.AppendNewBlock("network_interface", []string{})
	networkInterfaceBody := networkInterface.Body()
	networkInterfaceBody.SetAttributeTraversal("network", hcl.Traversal{
		hcl.TraverseRoot{
			Name: "google_compute_network",
		},
		hcl.TraverseAttr{
			Name: networkName,
		},
		hcl.TraverseAttr{
			Name: "id",
		},
	})
	accessConfig := networkInterfaceBody.AppendNewBlock("access_config", []string{})
	accessConfigBody := accessConfig.Body()
	accessConfigBody.SetAttributeTraversal("nat_ip", hcl.Traversal{
		hcl.TraverseRoot{
			Name: "google_compute_address",
		},
		hcl.TraverseAttr{
			Name: staticIPName,
		},
		hcl.TraverseAttr{
			Name: "address",
		},
	})

	bootDisk := gcpInstanceBody.AppendNewBlock("boot_disk", []string{})
	bootDiskBody := bootDisk.Body()
	initParams := bootDiskBody.AppendNewBlock("initialize_params", []string{})
	initParamsBody := initParams.Body()
	initParamsBody.SetAttributeValue("image", cty.StringVal(ami))

	gcpInstanceBody.SetAttributeValue("allow_stopping_for_update", cty.BoolVal(true))
}

// SetOutput adds output section in terraform state file so that we can call terraform output command and print instance_ip and instance_id to user
func SetOutput(rootBody *hclwrite.Body) {
	outputEip := rootBody.AppendNewBlock("output", []string{"instance_ips"})
	outputEipBody := outputEip.Body()
	outputEipBody.SetAttributeTraversal("value", hcl.Traversal{
		hcl.TraverseRoot{
			Name: "aws_eip",
		},
		hcl.TraverseAttr{
			Name: "myeip[*]",
		},
		hcl.TraverseAttr{
			Name: "public_ip",
		},
	})

	outputInstanceID := rootBody.AppendNewBlock("output", []string{"instance_ids"})
	outputInstanceIDBody := outputInstanceID.Body()
	outputInstanceIDBody.SetAttributeTraversal("value", hcl.Traversal{
		hcl.TraverseRoot{
			Name: "aws_instance",
		},
		hcl.TraverseAttr{
			Name: "aws_node[*]",
		},
		hcl.TraverseAttr{
			Name: "id",
		},
	})
}

// RunTerraform executes terraform apply function that creates the EC2 instances based on the .tf file provided
// returns a list of AWS node-IDs and node IPs
func RunTerraform(terraformDir string) ([]string, []string, error) {
	cmd := exec.Command(constants.Terraform, "init") //nolint:gosec
	cmd.Dir = terraformDir
	if err := cmd.Run(); err != nil {
		return nil, nil, err
	}
	cmd = exec.Command(constants.Terraform, "apply", "-auto-approve") //nolint:gosec
	cmd.Dir = terraformDir
	var stdBuffer bytes.Buffer
	var stderr bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd.Stdout = mw
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), constants.EIPLimitErr) {
			return nil, nil, errors.New(constants.EIPLimitErr)
		}
		return nil, nil, err
	}
	//instanceIDs, err := GetInstanceIDs(terraformDir)
	//if err != nil {
	//	return nil, nil, err
	//}
	//publicIPs, err := GetPublicIPs(terraformDir)
	//if err != nil {
	//	return nil, nil, err
	//}
	//return instanceIDs, publicIPs, nil
	return nil, nil, nil
}

//
//func GetInstanceIDs(terraformDir string) ([]string, error) {
//	cmd := exec.Command(constants.Terraform, "output", "instance_ids") //nolint:gosec
//	cmd.Dir = terraformDir
//	instanceIDsOutput, err := cmd.Output()
//	if err != nil {
//		return nil, err
//	}
//	instanceIDs := []string{}
//	instanceIDsOutputWoSpace := strings.TrimSpace(string(instanceIDsOutput))
//	// eip and nodeID outputs are bounded by [ and ,] , we need to remove them
//	trimmedInstanceIDs := instanceIDsOutputWoSpace[1 : len(instanceIDsOutputWoSpace)-3]
//	splitInstanceIDs := strings.Split(trimmedInstanceIDs, ",")
//	for _, instanceID := range splitInstanceIDs {
//		instanceIDWoSpace := strings.TrimSpace(instanceID)
//		// eip and nodeID both are bounded by double quotation "", we need to remove them before they can be used
//		instanceIDs = append(instanceIDs, instanceIDWoSpace[1:len(instanceIDWoSpace)-1])
//	}
//	return instanceIDs, nil
//}
//
//func GetPublicIPs(terraformDir string) ([]string, error) {
//	cmd := exec.Command(constants.Terraform, "output", "instance_ips") //nolint:gosec
//	cmd.Dir = terraformDir
//	eipsOutput, err := cmd.Output()
//	if err != nil {
//		return nil, err
//	}
//	publicIPs := []string{}
//	eipsOutputWoSpace := strings.TrimSpace(string(eipsOutput))
//	// eip and nodeID outputs are bounded by [ and ,] , we need to remove them
//	trimmedPublicIPs := eipsOutputWoSpace[1 : len(eipsOutputWoSpace)-3]
//	splitPublicIPs := strings.Split(trimmedPublicIPs, ",")
//	for _, publicIP := range splitPublicIPs {
//		publicIPWoSpace := strings.TrimSpace(publicIP)
//		// eip and nodeID both are bounded by double quotation "", we need to remove them before they can be used
//		publicIPs = append(publicIPs, publicIPWoSpace[1:len(publicIPWoSpace)-1])
//	}
//	return publicIPs, nil
//}
