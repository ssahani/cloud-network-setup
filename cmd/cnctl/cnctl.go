// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	cloudprovider "github.com/cloud-network-setup/pkg/cloudprovider"
	"github.com/cloud-network-setup/pkg/cloudprovider/azure"
	"github.com/cloud-network-setup/pkg/cloudprovider/ec2"
	"github.com/cloud-network-setup/pkg/conf"
	"github.com/cloud-network-setup/pkg/network"
	"github.com/cloud-network-setup/pkg/utils"
	"github.com/go-resty/resty/v2"
	"github.com/powersj/whatsthis"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func fetchCloudMetadata(url string) ([]byte, error) {
	client := resty.New()

	resp, err := client.R().Get(url)
	if resp.StatusCode() != 200 {
		fmt.Printf("Failed to fetch metadata: '%v'", resp.Error())
		return nil, err
	}

	return resp.Body(), nil
}

func displayAzureCloudNetworkMetadata(links *network.Links, n *azure.Azure) error {
	for i := 0; i < len(n.Network.Interface); i++ {
		subnet := n.Network.Interface[i].Ipv4.Subnet[0]

		var privateIp string
		var publicIp string
		for j := 0; j < len(n.Network.Interface[i].Ipv4.IPAddress); j++ {
			privateIp += fmt.Sprintf("%s/%s ", n.Network.Interface[i].Ipv4.IPAddress[j].PrivateIpAddress, subnet.Prefix)
			publicIp += fmt.Sprintf("%s ", n.Network.Interface[i].Ipv4.IPAddress[j].PublicIpAddress)
		}

		l, ok := links.LinksByMAC[strings.ToLower(utils.FormatTextToMAC(n.Network.Interface[i].MacAddress))]
		if !ok {
			continue
		}

		fmt.Printf("             Name: %+v \n", l.Name)
		fmt.Printf("      MAC Address: %+v \n", strings.ToLower(utils.FormatTextToMAC(n.Network.Interface[i].MacAddress)))
		fmt.Printf("        Public IP: %+v \n", publicIp)
		fmt.Printf("       Private IP: %+v \n", privateIp)
		fmt.Printf("           Subnet: %+v \n\n", subnet.Address)
	}

	return nil
}

func displayEC2CloudNetworkMetadata(l *network.Link, n *ec2.MAC) error {
	fmt.Printf("              Name: %+v \n", l.Name)
	fmt.Printf("       MAC Address: %+v \n", n.Mac)
	fmt.Printf("      Device Number: %+v \n", n.DeviceNumber)
	fmt.Printf("       Interface ID: %+v \n", n.InterfaceID)
	fmt.Printf("     Local Hostname: %+v \n", n.LocalHostname)
	fmt.Printf("        Local Ipv4S: %+v \n", n.LocalIpv4S)
	fmt.Printf("            OwnerID: %+v \n", n.OwnerID)

	if len(n.PublicHostname) > 0 {
		fmt.Printf("     PublicHostname: %+v \n", n.PublicHostname)
	}

	if len(n.PublicIpv4S) > 0 {
		fmt.Printf("        PublicIpv4S: %+v \n", n.PublicIpv4S)
	}

	if len(n.Ipv4Associations.Ipv4Association) > 0 {
		fmt.Printf("     pv4Association: %+v \n\n", n.Ipv4Associations.Ipv4Association)
	}

	fmt.Printf("    SecurityGroupId: %+v \n", n.SecurityGroupIds)
	fmt.Printf("     SecurityGroups: %+v \n", n.SecurityGroups)
	fmt.Printf("           SubnetID: %+v \n", n.SubnetID)
	fmt.Printf("SubnetIpv4CidrBlock: %+v \n", n.SubnetIpv4CidrBlock)
	fmt.Printf("              VpcID: %+v \n", n.VpcID)
	fmt.Printf("   VpcIpv4CidrBlock: %+v \n", n.VpcIpv4CidrBlock)
	fmt.Printf("  VpcIpv4CidrBlocks: %+v \n\n", n.VpcIpv4CidrBlocks)

	return nil
}

func fetchCloudNetworkMetadata() error {
	resp, err := fetchCloudMetadata("http://" + conf.IPFlag + ":" + conf.PortFlag + "/api/cloud/network")
	if err != nil {
		fmt.Printf("Failed to fetch instance metadata: '%+v'", err)
		return err
	}

	links, err := network.AcquireLinks()
	if err != nil {
		return err
	}

	provider, _ := whatsthis.Cloud()
	switch provider.Name {
	case cloudprovider.Azure:
		f := azure.Azure{}
		json.Unmarshal(resp, &f)

		displayAzureCloudNetworkMetadata(links, &f)
	case cloudprovider.AWS:
		m := make(map[string]interface{})
		json.Unmarshal(resp, &m)

		for _, v := range m {
			s, _ := json.Marshal(v)

			t := ec2.MAC{}
			json.Unmarshal(s, &t)

			l, ok := links.LinksByMAC[t.Mac]
			if !ok {
				continue
			}

			displayEC2CloudNetworkMetadata(&l, &t)
		}
	default:
		fmt.Printf("Unsupported cloud enviromennt: '%s'", provider)
	}

	return nil
}

func displayAzureCloudSystemMetadata(c *azure.Azure) {
	fmt.Printf("Azure Environment: %+v \n", c.Compute.AzEnvironment)
	fmt.Printf("         Location: %+v \n", c.Compute.Location)
	fmt.Printf("             Name: %+v \n", c.Compute.Name)
	fmt.Printf("          OS Type: %+v \n", c.Compute.OsType)
	fmt.Printf("            VM Id: %+v \n", c.Compute.VMID)
	if len(c.Compute.VMScaleSetName) > 0 {
		fmt.Printf("  VM ScaleSetName: %+v \n", c.Compute.VMScaleSetName)
	}
	fmt.Printf("          VM Size: %+v \n", c.Compute.VMSize)
	if len(c.Compute.Zone) > 0 {
		fmt.Printf("             Zone: %+v \n", c.Compute.Zone)
	}
	fmt.Printf("         Provider: %+v \n", c.Compute.Provider)
	fmt.Printf("  Subscription Id: %+v \n", c.Compute.SubscriptionID)
	if len(c.Compute.Publisher) > 0 {
		fmt.Printf("        Publisher: %+v \n", c.Compute.Publisher)
	}
	if len(c.Compute.Version) > 0 {
		fmt.Printf("         Version : %+v \n", c.Compute.Version)
	}
	if len(c.Compute.LicenseType) > 0 {
		fmt.Printf("     LicenseType : %+v \n", c.Compute.LicenseType)
	}
	fmt.Printf("     ComputerName: %+v \n", c.Compute.OsProfile.ComputerName)
	if len(c.Compute.SecurityProfile.SecureBootEnabled) > 0 {
		fmt.Printf("SecureBootEnabled: %+v \n", c.Compute.SecurityProfile.SecureBootEnabled)
	}
	if len(c.Compute.SecurityProfile.VirtualTpmEnabled) > 0 {
		fmt.Printf("VirtualTpmEnabled: %+v \n", c.Compute.SecurityProfile.VirtualTpmEnabled)
	}
	if len(c.Compute.TagsList) > 0 {
		fmt.Printf("         TagsList: %+v \n", c.Compute.TagsList)
	}
	if len(c.Compute.Plan.Name) > 0 {
		fmt.Printf("        Plan Name: %+v \n", c.Compute.Plan.Name)
	}
	if len(c.Compute.Plan.Name) > 0 {
		fmt.Printf("     Plan Product: %+v \n", c.Compute.Plan.Product)
	}
	if len(c.Compute.Plan.Publisher) > 0 {
		fmt.Printf("   Plan Publisher: %+v \n", c.Compute.Plan.Publisher)
	}
	if len(c.Compute.Offer) > 0 {
		fmt.Printf("            Offer: %+v \n", c.Compute.Offer)
	}
	if len(c.Compute.StorageProfile.DataDisks) > 0 {
		fmt.Printf("       StorageProfile: %+v \n", c.Compute.StorageProfile.DataDisks)
	}
	fmt.Printf("    AdminUsername: %+v \n", c.Compute.OsProfile.AdminUsername)
	fmt.Printf("      Public Keys: %+v \n\n", c.Compute.PublicKeys)
}

func displayEC2CloudSystemMetadata(c *ec2.EC2) {
	fmt.Printf("             AmiID: %+v \n", c.AmiID)
	fmt.Printf("          Location: %+v \n", c.AmiLaunchIndex)
	fmt.Printf("   AmiManifestPath: %+v \n", c.AmiManifestPath)
	fmt.Printf("BlockDeviceMapping: %+v \n", c.BlockDeviceMapping)
	fmt.Printf("          Hostname: %+v \n", c.Hostname)
	fmt.Printf("    PublicHostname: %+v \n", c.PublicHostname)
	fmt.Printf("     LocalHostname: %+v \n", c.LocalHostname)
	fmt.Printf("    InstanceAction: %+v \n", c.InstanceAction)
	fmt.Printf("        InstanceID: %+v \n", c.InstanceID)
	fmt.Printf(" InstanceLifeCycle: %+v \n", c.InstanceLifeCycle)
	fmt.Printf("      InstanceType: %+v \n", c.InstanceType)
	fmt.Printf("         Placement: %+v \n", c.Placement)
	fmt.Printf("           Profile: %+v \n", c.Profile)
	fmt.Printf("       Mac Address: %+v \n", c.Mac)
	fmt.Printf("         LocalIpv4: %+v \n", c.LocalIpv4)
	fmt.Printf("        PublicIpv4: %+v \n", c.PublicIpv4)
	fmt.Printf("   Services Domain: %+v \n", c.Services.Domain)
	fmt.Printf("Services Partition: %+v \n", c.Services.Partition)
	fmt.Printf("        PublicKeys: %+v \n\n", c.PublicKeys)
}

func fetchCloudSystemMetadata() {
	resp, err := fetchCloudMetadata("http://" + conf.IPFlag + ":" + conf.PortFlag + "/api/cloud/system")
	if err != nil {
		return
	}

	provider, err := whatsthis.Cloud()
	switch provider.Name {
	case cloudprovider.Azure:
		f := azure.Azure{}
		json.Unmarshal(resp, &f)

		displayAzureCloudSystemMetadata(&f)
	case cloudprovider.AWS:
		f := ec2.EC2{}
		json.Unmarshal(resp, &f)

		displayEC2CloudSystemMetadata(&f)
	default:
		fmt.Printf("Failed to detect cloud enviroment: '%+v'", err)
		return
	}
}

func main() {
	conf.Parse()
	log.SetOutput(ioutil.Discard)

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s\n", c.App.Version)
	}

	app := &cli.App{
		Name:     "cpctl",
		Version:  "v0.1",
		HelpName: "Introspects cloud network metadata",
	}
	app.EnableBashCompletion = true
	app.Commands = []*cli.Command{
		{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "Show status system cloud metadata",
			Action: func(c *cli.Context) error {
				switch c.Args().First() {
				case "system":
					fetchCloudSystemMetadata()
				case "network":
					fetchCloudNetworkMetadata()
				default:
					fetchCloudSystemMetadata()
					fetchCloudNetworkMetadata()
				}
				return nil
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("Failed to run cli: '%v'", err)
	}
}
