// +build solaris illumos

package water

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// from <stropts.h>
type strioctl struct {
	ic_cmd    int32
	ic_timout int32
	ic_len    int32
	ic_dp     unsafe.Pointer
}

type lifreq struct {
	lfr_name    [32]byte
	lifru_ppa   uint32
	lifru_flags uint64
}

const (
	IOCTL        = 54
	TUNNEWPPA    = (('T' << 16) | 0x0001)
	TUNSETPPA    = (('T' << 16) | 0x0002)
	I_STR        = (('S' << 8) | 010)
	I_PUSH       = (('S' << 8) | 002)
	SIOCSLIFNAME = 3229116801 // #define	SIOCSLIFNAME	_IOWR('i', 129, struct lifreq)
	IFF_IPV6     = 0x2000000
)

func openDev(config Config) (ifce *Interface, err error) {
	switch config.Name[:8] {
	case "/dev/tun":
		return newTUN(config)
	default:
		return nil, fmt.Errorf("unsupported interface type")
	}
}

func newTUN(config Config) (ifce *Interface, err error) {
	fmt.Println("I_PUSH:", I_PUSH)

	tunfd, tunfderr := os.OpenFile("/dev/tun", os.O_RDWR, 0)
	tun2fd, tun2fderr := os.OpenFile("/dev/tun", os.O_RDWR, 0)
	if tunfderr != nil || tun2fderr != nil {
		return nil, fmt.Errorf("error opening /dev/tun: %v %v", tunfderr, tun2fderr)
	}

	ipfd, iperr := os.OpenFile("/dev/ip6", os.O_RDWR, 0)
	if iperr != nil {
		return nil, iperr
	}

	var ppa int
	var ioctlaction int

	fmt.Println("Configured device name is", config.Name)

	if len(config.Name) > 8 {
		fmt.Println("Try to set PPA")
		ppa, err = strconv.Atoi(config.Name[8:])
		if err != nil {
			return nil, err
		}
		ioctlaction = TUNSETPPA
	} else {
		fmt.Println("Try to new PPA")
		ioctlaction, ppa = TUNNEWPPA, 0
	}

	fmt.Println("Trying TUNNEWPPA/TUNSETPPA", uintptr(ppa))
	_, _, errno := syscall.Syscall(IOCTL, tunfd.Fd(), uintptr(ioctlaction), uintptr(ppa))
	if errno != 0 {
		return nil, fmt.Errorf("ioctl(TUNNEWPPA/TUNSETPPA): %v", errno)
	}

	/*
		fmt.Println("Trying push IP stack to tun2fd")
		stack := []byte("ip")
		_, _, errno = syscall.Syscall(IOCTL, tun2fd.Fd(), I_PUSH, uintptr(unsafe.Pointer(&stack)))
		if errno != 0 {
			return nil, fmt.Errorf("ioctl(I_PUSH): %v", errno)
		}
	*/

	ifr := lifreq{
		lifru_ppa:   uint32(ppa),
		lifru_flags: IFF_IPV6,
	}
	copy(ifr.lfr_name[:], fmt.Sprintf("tun%d", ppa))
	fmt.Println("Trying set interface name to", string(ifr.lfr_name[:]), "for PPA", uint32(ppa))
	_, _, errno = syscall.Syscall(IOCTL, tun2fd.Fd(), SIOCSLIFNAME, uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		return nil, fmt.Errorf("ioctl(SIOCSLIFNAME): %v", errno)
	}

	_ = ipfd

	fmt.Println("The selected PPA is", ppa)

	intfd := os.NewFile(uintptr(ppa), "pipe")
	if intfd == nil {
		return nil, fmt.Errorf("invalid intfd")
	}

	/*
			154     if((if_fd = open(device, O_RDWR, 0)) < 0) {
		  155         logger(LOG_ERR, "Could not open %s: %s\n", device, strerror(errno));
		  156         return false;
		  157     }
		  158
		  159     if(ioctl(if_fd, I_PUSH, "ip") < 0) {
		  160         logger(LOG_ERR, "Could not push IP module onto %s %s!", device_info, device);
		  161         return false;
		  162     }
	*/

	// ifce = &Interface{isTAP: false, ReadWriteCloser: fd, name: config.Name[5:]}
	return
}
