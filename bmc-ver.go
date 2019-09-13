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
	{"ubo", "worktrees/u-boot/platina-mk1-bmc", "platina-mk1-bmc-ubo.bin"},
	{"dtb", "worktrees/linux/platina-mk1-bmc", "platina-mk1-bmc-dtb.bin"},
	{"env", ".", "platina-mk1-bmc-env.bin"},
	{"ker", "worktrees/linux/platina-mk1-bmc", "platina-mk1-bmc.vmlinuz"},
	{"itb", "../goes-bmc", "platina-mk1-bmc-itb.bin"},
}
var ImgInfo [5]IMGINFO

func makeVer(k string) {
	Release := getReleaseInfo(k)
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
		panic("Error dev or rel not found")
	}
	return kk
}

func getImageInfo(x int, nm string, di string, im string) {
	u, err := exec.Command("ls", "-l", im).Output()
	if err != nil {
		fmt.Printf("Error %s on ls -l %s\n", err, im)
		panic(err)
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
			panic(err)
		}
		err = os.Chdir(di)
		if err != nil {
			panic(err)
		}
		defer os.Chdir(od)
		u, err = exec.Command("git", "describe").Output()
		if err != nil {
			fmt.Printf("In directory %s (od=%s): %s\n", di, od, err)
			fmt.Println(u)
			panic(err)
		}
		uu = strings.Split(string(u), "\n")
		ImgInfo[x].Tag = (uu[0])
		u, err = exec.Command("git", "log", "-1").Output()
		if err != nil {
			panic(err)
		}
		uu = strings.Split(string(u), "\n")
		uuu := strings.Split(string(uu[0]), " ")
		ImgInfo[x].Commit = (uuu[1])
		err = os.Chdir(od)
		if err != nil {
			panic(err)
		}
	}
	u, err = exec.Command("sha1sum", im).Output()
	if err != nil {
		panic(err)
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
		panic(err)
	}
}
