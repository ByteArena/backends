package vm

import (
	"errors"
	"fmt"
	"log"
	"syscall"

	"github.com/hashicorp/errwrap"
	"github.com/rkt/rkt/networking/tuntap"
	"github.com/vishvananda/netlink"
)

func setupTapDevice(name string) (netlink.Link, error) {

	// network device names are limited to 16 characters
	// the suffix %d will be replaced by the kernel with a suitable number
	nameTemplate := fmt.Sprintf("ba-%s-tap%%d", name[0:4])
	ifName, err := tuntap.CreatePersistentIface(nameTemplate, tuntap.Tap)
	if err != nil {
		log.Println(err)
		return nil, errwrap.Wrap(errors.New("tuntap persist"), err)
	}

	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return nil, errwrap.Wrap(fmt.Errorf("cannot find link %q", ifName), err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return nil, errwrap.Wrap(fmt.Errorf("cannot set link up %q", ifName), err)
	}

	return link, nil
}

func ensureBridgeIsUp(brName string, mtu int) (*netlink.Bridge, error) {
	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: brName,
			MTU:  mtu,
		},
	}

	if err := netlink.LinkAdd(br); err != nil {
		if err != syscall.EEXIST {
			return nil, errwrap.Wrap(fmt.Errorf("could not add %q", brName), err)
		}

		// it's ok if the device already exists as long as config is similar
		br, err = bridgeByName(brName)
		if err != nil {
			return nil, err
		}
	}

	if err := netlink.LinkSetUp(br); err != nil {
		return nil, err
	}

	return br, nil
}

func bridgeByName(name string) (*netlink.Bridge, error) {
	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, errwrap.Wrap(fmt.Errorf("could not lookup %q", name), err)
	}
	br, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("%q already exists but is not a bridge", name)
	}
	return br, nil
}
