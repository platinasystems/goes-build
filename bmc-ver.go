// Copyright © 2015-2017 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

type IMAGE struct {
	Name string
	Dir  string
	File string
}
type IMGINFO struct {
	Name   string
	Build  string
	User   string
	Size   string
	Tag    string
	Commit string
	Chksum string
}

var Images = [5]IMAGE{
	{"ubo", "src/u-boot", "platina-mk1-bmc-ubo.bin"},
	{"dtb", "src/linux", "platina-mk1-bmc.dtb"},
	{"env", ".", "platina-mk1-bmc-env.bin"},
	{"ker", "src/linux", "platina-mk1-bmc.vmlinuz"},
	{"ini", "../go", "initrd.img.xz"},
}
var ImgInfo [5]IMGINFO

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error no args")
		os.Exit(1)
	}
	Release := getReleaseInfo(os.Args[1])
	for i, _ := range Images {
		getImageInfo(i, Images[i].Name, Images[i].Dir, Images[i].File)
	}
	writeVerFile(Release)
}

func getReleaseInfo(k string) string {
	t := time.Now()
	kk := ""
	switch k {
	case "dev":
		kk = "dev"
	case "rel":
		kk = t.Format("20060102")
	default:
		fmt.Println("Error dev or rel not found")
		os.Exit(1)
	}
	return kk
}

func getImageInfo(x int, nm string, di string, im string) {
	u, err := exec.Command("ls", "-l", im).Output()
	if err != nil {
		fmt.Println("Error ls")
		os.Exit(1)
	}
	v := strings.Replace(string(u), "  ", " ", -1)
	v = strings.Replace(v, "  ", " ", -1)
	v = strings.Replace(v, "  ", " ", -1)
	v = strings.Replace(v, "  ", " ", -1)
	uu := strings.Split(v, " ")
	t := time.Now()
	yr := t.Format("2006")
	ImgInfo[x].Name = "platina-mk1-bmc-" + nm + ".bin"
	ImgInfo[x].Build = uu[5] + " " + uu[6] + " " + yr + " " + uu[7]
	ImgInfo[x].User = uu[2]
	ImgInfo[x].Size = uu[4]

	{
		od, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting working directory")
			os.Exit(1)
		}
		err = os.Chdir(di)
		if err != nil {
			fmt.Println("Error changing directory")
			os.Exit(1)
		}
		defer os.Chdir(od)
		u, err = exec.Command("git", "describe", "--abbrev=0").Output()
		if err != nil {
			fmt.Println("Error running git describe")
			os.Exit(1)
		}
		uu = strings.Split(string(u), "\n")
		ImgInfo[x].Tag = (uu[0])
		u, err = exec.Command("git", "log", "-1").Output()
		if err != nil {
			fmt.Println("Error running git log")
			os.Exit(1)
		}
		uu = strings.Split(string(u), "\n")
		uuu := strings.Split(string(uu[0]), " ")
		ImgInfo[x].Commit = (uuu[1])
		err = os.Chdir(od)
		if err != nil {
			fmt.Println("Error restoring working directory")
			os.Exit(1)
		}
	}
	u, err = exec.Command("sha1sum", im).Output()
	if err != nil {
		fmt.Println("Error running sha1sum")
		os.Exit(1)
	}
	uu = strings.Split(string(u), "\n")
	uuu := strings.Split(string(uu[0]), " ")
	ImgInfo[x].Chksum = (uuu[0])
}

func writeVerFile(Release string) {
	VerBlock := make([]byte, 256*1024)
	for i, _ := range VerBlock {
		VerBlock[i] = 0xff
	}
	copy(VerBlock[0x00:], Release)
	jsonInfo, _ := json.Marshal(ImgInfo)
	copy(VerBlock[0x100:], jsonInfo)
	err := ioutil.WriteFile("platina-mk1-bmc-ver.bin", VerBlock, 0644)
	if err != nil {
		fmt.Println("Error writing version file")
		os.Exit(1)
	}
}
