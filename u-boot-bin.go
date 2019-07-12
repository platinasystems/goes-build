package main

import (
	"fmt"
	"io/ioutil"
)

// File size is 768K bytes

// [0] first 2 * 512 (1024) bytes unused
// [1024] then 512 bytes from qspi-header-sclk00
// [1536] then 5 * 512 (2560) bytes of zero
// [4096] then U-boot

const ubootSize = 768 * 1024

const headerStart = 2 * 512

const ubootStart = 8 * 512

func makeUboot(ubo string) []byte {
	ubootbin := make([]byte, ubootSize)

	if uboot, err := ioutil.ReadFile(ubo); err != nil {
		fmt.Printf("Unable to read %s: %s\n", ubo, err)
		panic(err)
	} else {
		if len(uboot) > ubootSize-ubootStart {
			panic(fmt.Errorf("U-boot size of %d exceeds max %d\n",
				len(uboot), ubootSize-ubootStart))
		}
		copy(ubootbin[ubootStart:], uboot)
	}

	if header, err := ioutil.ReadFile("qspi-header-sckl00"); err != nil {
		fmt.Printf("Unable to read qspi-header-sclk00: %s\n", err)
		panic(err)
	} else {
		copy(ubootbin[headerStart:], header)
	}
	return ubootbin
}
