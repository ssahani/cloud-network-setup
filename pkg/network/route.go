// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"errors"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

func GetDefaultIpv4Gateway() (string, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		if route.Dst == nil || route.Dst.String() == "0.0.0.0/0" {
			if route.Gw.To4() == nil {
				return "", errors.New("failed to find gateway, default route is present")
			}

			return route.Gw.To4().String(), nil
		}
	}

	return "", errors.New("not found")
}

func GetDefaultIpv4GatewayByLink(ifIndex int) (string, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		if route.Dst == nil || route.Dst.String() == "0.0.0.0/0" {
			if route.LinkIndex == ifIndex {
				return route.Gw.To4().String(), nil
			}
		}
	}

	return "", errors.New("not found")
}

func AddRoute(ifIndex int, table int, gateway string) error {
	gw := net.ParseIP(gateway).To4()

	rt := netlink.Route{
		LinkIndex: ifIndex,
		Gw:        gw,
		Table:     table,
	}

	if err := netlink.RouteAdd(&rt); err != nil && err.Error() != "file exists" {
		return err
	}

	return nil
}
