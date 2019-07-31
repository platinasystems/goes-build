// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

// build goes machine(s)
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/platinasystems/go-cpio"
)

const (
	platina               = ".."
	platinaFe1            = platina + "/fe1"
	platinaFe1Firmware    = platina + "/firmware-fe1a"
	platinaGoes           = platina + "/goes"
	platinaGoesLegacy     = platina + "/goes-legacy"
	platinaGoesLegacyMain = platinaGoesLegacy + "/main"
	platinaSecrets        = platina + "/platina-secrets"
	platinaVnetMk1        = platina + "/vnet-platina-mk1"

	platinaSystemBuildSrc = platina + "/system-build/src"

	platinaGoesMainIP                   = platinaGoesLegacyMain + "/ip"
	platinaGoesMainGoesPrefix           = platinaGoesLegacyMain + "goes-"
	platinaGoesMainGoesExample          = platina + "/goes-example"
	platinaGoesMainGoesBoot             = platina + "/goes-boot"
	platinaGoesMainGoesInstaller        = platinaGoesLegacyMain + "/goes-installer"
	platinaGoesMainGoesPlatinaMk1       = platina + "/goes-platina-mk1"
	platinaGoesMainGoesPlatinaMk1Bmc    = platina + "/goes-bmc"
	platinaGoesMainGoesPlatinaMk2       = platinaGoesLegacyMain + "/goes-platina-mk2"
	platinaGoesMainGoesPlatinaMk2Lc1Bmc = platinaGoesMainGoesPlatinaMk2 + "-lc1-bmc"
	platinaGoesMainGoesPlatinaMk2Mc1Bmc = platinaGoesMainGoesPlatinaMk2 + "-mc1-bmc"
)

type target struct {
	name         string
	maker        func(tg *target) error
	main         string
	dir          string
	def          bool
	dependencies []*target
	once         sync.Once
}

type goenv struct {
	goarch           string
	goos             string
	gnuPrefix        string
	kernelMakeTarget string
	kernelPath       string
	kernelConfigPath string
	kernelArch       string
	boot             string
	cpioSuffix       string
	cpioTrimPrefix   string
}

var (
	goarchFlag = flag.String("goarch", runtime.GOARCH,
		"GOARCH of PACKAGE build")
	goosFlag = flag.String("goos", runtime.GOOS,
		"GOOS of PACKAGE build")
	cloneFlag = flag.Bool("clone", false,
		"Fallback to 'git clone' if git worktree does not work.")
	legacyFlag = flag.Bool("legacy", false,
		"Use legacy flash layout.")
	nFlag = flag.Bool("n", false,
		"print 'go build' commands but do not run them.")
	oFlag    = flag.String("o", "", "output file name of PACKAGE build")
	rFlag    = flag.String("r", "", "rebase worktrees before build")
	tagsFlag = flag.String("tags", "", `
debug	disable optimizer and increase vnet log
diag	include manufacturing diagnostics with BMC
`)
	xFlag = flag.Bool("x", false, "print 'go build' commands.")
	vFlag = flag.Bool("v", false,
		"print the names of packages as they are compiled.")
	zFlag = flag.Bool("z", false, "print 'goes-build' commands.")
	host  = goenv{
		goarch: runtime.GOARCH,
		goos:   runtime.GOOS,
	}
	amd64Linux = goenv{
		goarch:           "amd64",
		goos:             "linux",
		gnuPrefix:        "x86_64-linux-gnu-",
		kernelMakeTarget: "bindeb-pkg",
		kernelPath:       "arch/x86/boot/bzImage",
		kernelConfigPath: "arch/x86/configs",
		kernelArch:       "x86_64",
		boot:             "coreboot",
		cpioSuffix:       ".cpio.xz",
	}
	armLinux = goenv{
		goarch:           "arm",
		goos:             "linux",
		gnuPrefix:        "arm-linux-gnueabi-",
		kernelMakeTarget: "zImage dtbs",
		kernelPath:       "arch/arm/boot/zImage",
		kernelConfigPath: "arch/arm/configs",
		kernelArch:       "arm",
		boot:             "u-boot",
		cpioSuffix:       ".cpio.xz",
		cpioTrimPrefix:   "goes-",
	}

	corebootExampleAmd64Config  = "example-amd64_defconfig"
	corebootExampleAmd64Machine = "example-amd64"

	corebootPlatinaMk1Config  = "platina-mk1_defconfig"
	corebootPlatinaMk1Machine = "platina-mk1"

	corebootExampleAmd64    *target
	corebootExampleAmd64Rom *target
	corebootPlatinaMk1      *target
	corebootPlatinaMk1Rom   *target
	exampleAmd64Vmlinuz     *target
	goesBoot                *target
	goesBootArm             *target
	goesExample             *target
	goesExampleArm          *target
	goesIP                  *target
	goesIPTest              *target
	goesPlatinaMk1          *target
	goesPlatinaMk1Bmc       *target
	goesPlatinaMk1Installer *target
	goesPlatinaMk1Test      *target
	goesPlatinaMk2Lc1Bmc    *target
	goesPlatinaMk2Mc1Bmc    *target
	itbPlatinaMk1Bmc        *target
	platinaMk1BmcVmlinuz    *target
	platinaMk1Vmlinuz       *target
	platinaMk2Lc1BmcVmlinuz *target
	platinaMk2Mc1BmcVmlinuz *target
	ubootPlatinaMk1Bmc      *target
	vnetPlatinaMk1          *target
	zipPlatinaMk1Bmc        *target

	allTargets = []*target{}
	targetMap  = map[string]*target{}
)

func init() {
	flag.Usage = usage

	corebootExampleAmd64 = &target{
		name:  "coreboot-example-amd64",
		maker: makeAmd64Boot,
		main:  corebootExampleAmd64Config,
	}

	corebootExampleAmd64Rom = &target{
		name:  "coreboot-example-amd64.rom",
		maker: makeAmd64CorebootRom,
		main:  corebootExampleAmd64Machine,
		def:   true,
	}

	corebootPlatinaMk1 = &target{
		name:  "coreboot-platina-mk1",
		maker: makeAmd64Boot,
		main:  corebootPlatinaMk1Config,
	}

	corebootPlatinaMk1Rom = &target{
		name:  "coreboot-platina-mk1.rom",
		maker: makeAmd64CorebootRom,
		main:  corebootPlatinaMk1Machine,
		def:   true,
	}

	exampleAmd64Vmlinuz = &target{
		name:  "example-amd64.vmlinuz",
		maker: makeAmd64LinuxKernel,
		main:  "platina-example-amd64_defconfig",
		def:   true,
	}

	goesBoot = &target{
		name:  "goes-boot",
		maker: makeAmd64LinuxInitramfs,
		main:  platinaGoesMainGoesBoot,
		dir:   "../goes-boot",
	}

	goesBootArm = &target{
		name:  "goes-boot-arm",
		maker: makeArmLinuxInitramfs,
		main:  platinaGoesMainGoesBoot,
		dir:   "../goes-boot",
	}

	goesExample = &target{
		name:  "goes-example",
		maker: makeHost,
		main:  platinaGoesMainGoesExample,
		dir:   "../goes-example",
		def:   true,
	}

	goesExampleArm = &target{
		name:  "goes-example-arm",
		maker: makeArmLinuxStatic,
		main:  platinaGoesMainGoesExample,
		dir:   "../goes-example",
		def:   true,
	}

	goesIP = &target{
		name:  "goes-ip",
		maker: makeHost,
		main:  platinaGoesMainIP,
	}

	goesIPTest = &target{
		name:  "goes-ip.test",
		maker: makeHostTest,
		main:  platinaGoesMainIP,
	}

	goesPlatinaMk1 = &target{
		name:  "goes-platina-mk1",
		maker: makeGoesPlatinaMk1,
		main:  platinaGoesMainGoesPlatinaMk1,
		dir:   "../goes-platina-mk1",
		def:   true,
	}

	goesPlatinaMk1Bmc = &target{
		name:  "goes-platina-mk1-bmc",
		maker: makeArmLinuxInitramfs,
		main:  platinaGoesMainGoesPlatinaMk1Bmc,
		dir:   "../goes-bmc",
	}

	goesPlatinaMk1Installer = &target{
		name:  "goes-platina-mk1-installer",
		maker: makeGoesPlatinaMk1Installer,
		main:  platinaGoesMainGoesPlatinaMk1,
	}

	goesPlatinaMk1Test = &target{
		name:  "goes-platina-mk1.test",
		maker: makeAmd64LinuxTest,
		main:  platinaGoesMainGoesPlatinaMk1,
	}

	goesPlatinaMk2Lc1Bmc = &target{
		name:  "goes-platina-mk2-lc1-bmc",
		maker: makeArmLinuxStatic,
		main:  platinaGoesMainGoesPlatinaMk2Lc1Bmc,
	}

	goesPlatinaMk2Mc1Bmc = &target{
		name:  "goes-platina-mk2-mc1-bmc",
		maker: makeArmLinuxStatic,
		main:  platinaGoesMainGoesPlatinaMk2Mc1Bmc,
	}

	itbPlatinaMk1Bmc = &target{
		name:  "platina-mk1-bmc.itb",
		maker: makeArmItb,
		main:  "platina-mk1-bmc.its",
	}

	platinaMk1BmcVmlinuz = &target{
		name:  "platina-mk1-bmc.vmlinuz",
		maker: makeArmLinuxKernel,
		main:  "platina-mk1-bmc_defconfig",
	}

	platinaMk1Vmlinuz = &target{
		name:  "platina-mk1.vmlinuz",
		maker: makeAmd64LinuxKernel,
		main:  "platina-mk1_defconfig",
	}

	platinaMk2Lc1BmcVmlinuz = &target{
		name:  "platina-mk2-lc1-bmc.vmlinuz",
		maker: makeArmLinuxKernel,
		main:  "platina-mk2-lc1-bmc_defconfig",
	}

	platinaMk2Mc1BmcVmlinuz = &target{
		name:  "platina-mk2-mc1-bmc.vmlinuz",
		maker: makeArmLinuxKernel,
		main:  "platina-mk2-mc1-bmc_defconfig",
	}

	ubootPlatinaMk1Bmc = &target{
		name:  "u-boot-platina-mk1-bmc",
		maker: makeArmBoot,
		main:  "platinamx6boards_qspi_defconfig",
	}

	vnetPlatinaMk1 = &target{
		name:  "vnet-platina-mk1",
		maker: makeAmd64LinuxStatic,
		main:  platinaVnetMk1,
		dir:   "../vnet-platina-mk1",
		def:   true,
	}

	zipPlatinaMk1Bmc = &target{
		name:  "platina-mk1-bmc.zip",
		maker: makeArmZipfile,
		def:   true,
	}

	// Set up dependencies. We have to do this after we have set up all
	// of the targets, since we need the pointer to the target already
	// set up.

	corebootExampleAmd64Rom.dependencies = []*target{
		corebootExampleAmd64,
		exampleAmd64Vmlinuz,
		goesBoot,
	}

	corebootPlatinaMk1Rom.dependencies = []*target{
		corebootPlatinaMk1,
		platinaMk1Vmlinuz,
		goesBoot,
	}

	itbPlatinaMk1Bmc.dependencies = []*target{
		goesPlatinaMk1Bmc,
		platinaMk1BmcVmlinuz,
	}

	zipPlatinaMk1Bmc.dependencies = []*target{
		itbPlatinaMk1Bmc,
		ubootPlatinaMk1Bmc,
	}

	// Set up the list of targets

	allTargets = []*target{
		corebootExampleAmd64,
		corebootExampleAmd64Rom,
		corebootPlatinaMk1,
		corebootPlatinaMk1Rom,
		exampleAmd64Vmlinuz,
		goesBoot,
		goesBootArm,
		goesExample,
		goesExampleArm,
		goesIP,
		goesIPTest,
		goesPlatinaMk1,
		goesPlatinaMk1Bmc,
		goesPlatinaMk1Installer,
		goesPlatinaMk1Test,
		goesPlatinaMk2Lc1Bmc,
		goesPlatinaMk2Mc1Bmc,
		itbPlatinaMk1Bmc,
		platinaMk1BmcVmlinuz,
		platinaMk1Vmlinuz,
		platinaMk2Lc1BmcVmlinuz,
		platinaMk2Mc1BmcVmlinuz,
		ubootPlatinaMk1Bmc,
		vnetPlatinaMk1,
		zipPlatinaMk1Bmc,
	}
	for _, t := range allTargets {
		if _, p := targetMap[t.name]; p {
			panic("Duplicate target " + t.name)
		}
		targetMap[t.name] = t
	}
}

func makeTargets(parent string, targets []*target) {
	var wg sync.WaitGroup

	for _, tg := range targets {
		tg.once.Do(func() {
			if parent == "" {
				fmt.Printf("# Making Package %s\n", tg.name)
			} else {
				fmt.Printf("# Making dependent package %s for %s\n",
					tg.name, parent)
			}
			makeTargets(tg.name, tg.dependencies)

			wg.Add(1)
			go func(tg *target, wg *sync.WaitGroup) {
				err := tg.maker(tg)
				if err != nil {
					fmt.Printf("Error making package %s\n", tg.name)
					panic(err)
				}
				if parent == "" {
					fmt.Printf("# Done making Package %s\n", tg.name)
				} else {
					fmt.Printf("# Done making dependent package %s for %s\n",
						tg.name, parent)
				}
				wg.Done()
			}(tg, &wg)
		})
	}
	wg.Wait()
}

func main() {
	flag.Parse()
	targetsReq := flag.Args()
	tgs := make([]*target, 0)
	if len(targetsReq) == 0 {
		for _, t := range allTargets {
			if t.def {
				tgs = append(tgs, t)
			}
		}
	} else if targetsReq[0] == "all" {
		tgs = allTargets
	} else {
		for _, t := range targetsReq {
			if tg, p := targetMap[t]; p {
				tgs = append(tgs, tg)
			} else {
				panic("Unknown target " + t)
			}
		}
	}
	makeTargets("", tgs)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:", os.Args[0],
		"[ OPTION... ] [ TARGET... | PACKAGE ]")
	fmt.Fprintln(os.Stderr, "\nOptions:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nDefault Targets:")
	for _, t := range allTargets {
		if t.def {
			fmt.Fprint(os.Stderr, "\t", t.name, "\n")
		}
	}
	fmt.Fprintln(os.Stderr, "\n\"all\" Targets:")
	for _, t := range allTargets {
		fmt.Fprint(os.Stderr, "\t", t.name, "\n")
	}
}

func makeArmLinuxStatic(tg *target) error {
	return armLinux.godoforpkg(tg, "build", "-o", tg.name, "-tags", "netgo",
		"-ldflags", "-d", tg.main)
}

func makeArmBoot(tg *target) (err error) {
	machine := strings.TrimPrefix(tg.name, "u-boot-")
	if err = armLinux.makeboot(tg.name, "make "+tg.main); err != nil {
		return err
	}
	env, err := makeUbootEnv()
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(machine+"-env.bin", env, 0644); err != nil {
		return err
	}
	uboot := makeUboot("worktrees/u-boot/" + machine + "/u-boot-dtb.imx")
	if err = ioutil.WriteFile(machine+"-ubo.bin", uboot, 0644); err != nil {
		return err
	}

	return nil
}

func makeArmItb(tg *target) (err error) {
	machine := strings.TrimSuffix(tg.name, ".itb")

	cmdline := "mkimage -f goes-bmc.its " + machine + "-itb.bin"
	err = shellCommandRun(cmdline)
	if err != nil {
		return
	}

	s, err := os.Stat(machine + "-itb.bin")
	if err != nil {
		return
	}
	limit := int64(0x00800000)
	kind := ""
	if *legacyFlag {
		limit = 0x00500000
		kind = "legacy "
	}
	if s.Size() > limit {
		return fmt.Errorf("ITB size of %d exceeds %slimit of %d",
			s.Size(), kind, limit)
	}
	return
}

func makeArmZipfile(tg *target) (err error) {
	machine := strings.TrimSuffix(tg.name, ".zip")

	makeVer("rel") // FIXME

	zipFile, err := os.Create(machine + ".zip")
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	type filemap struct {
		in     string
		out    string
		offset int64
		len    int64
	}

	fileMaps := []filemap{
		{in: "-ubo.bin", offset: 0x0, len: 0x80000},
		{in: "-ubo.bin", out: "-dtb.bin", offset: 0x80000, len: 0x40000},
		{in: "-env.bin"},
		{in: "-ver.bin"},
	}

	if *legacyFlag {
		fileMaps = append(fileMaps, filemap{in: "-itb.bin",
			out: "-ker.bin", offset: 0x0, len: 0x200000})
		fileMaps = append(fileMaps, filemap{in: "-itb.bin",
			out: "-ini.bin", offset: 0x200000, len: 0x300000})
	} else {
		fileMaps = append(fileMaps, filemap{in: "-itb.bin"})
	}

	for _, fileMap := range fileMaps {
		file, err := os.Open(machine + fileMap.in)
		if err != nil {
			fmt.Printf("Error opening %s: %s\n", machine+fileMap.out,
				err)
			os.Remove(machine + ".zip")
			panic(err)
		}
		defer file.Close()

		// Get the file information
		info, err := file.Stat()
		if err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}

		if fileMap.offset != 0 && info.Size() <= fileMap.offset {
			fmt.Printf("Skipping %s offset %d greater than length %d\n",
				machine+fileMap.in, fileMap.offset, info.Size())
			continue
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}
		if fileMap.out != "" {
			header.Name = machine + fileMap.out
		}

		len := info.Size() - fileMap.offset
		if fileMap.len != 0 && fileMap.len < len {
			len = fileMap.len
		}

		// Change to deflate to gain better compression
		// see http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}
		off, err := file.Seek(fileMap.offset, io.SeekStart)
		if err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}
		if off != fileMap.offset {
			os.Remove(machine + ".zip")
			panic(fmt.Errorf("Seek to %d failed - got %d",
				fileMap.offset, off))
		}
		written, err := io.CopyN(writer, file, len)
		if err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}
		if written != len {
			os.Remove(machine + ".zip")
			panic(fmt.Errorf("Expected to write %d but wrote %d",
				len, written))
		}
		armLinux.log("added", header.Name, "to", machine+".zip")
	}
	fh := &zip.FileHeader{Name: machine + "-v2", Modified: time.Now()}
	_, err = zipWriter.CreateHeader(fh)
	if err != nil {
		os.Remove(machine + ".zip")
		panic(err)
	}
	armLinux.log("added", fh.Name, "to", machine+".zip")

	return nil
}

func makeArmLinuxKernel(tg *target) (err error) {
	machine := strings.TrimSuffix(tg.name, ".vmlinuz")
	err = armLinux.makeLinux(tg.name, tg.main)
	if err != nil {
		return
	}
	dtb := "worktrees/linux/" + machine + "/arch/arm/boot/dts/" + machine + ".dtb"
	cmdline := "cp " + dtb + " " + machine + "-dtb.bin"
	if err = shellCommandRun(cmdline); err != nil {
		return err
	}
	return
}

func makeArmLinuxInitramfs(tg *target) (err error) {
	machine := strings.TrimPrefix(tg.name, "goes-")
	machine = strings.TrimSuffix(machine, ".cpio.xz")
	err = makeArmLinuxStatic(tg)
	if err != nil {
		return
	}
	err = armLinux.makeCpioArchive(tg)

	return
}

func makeAmd64Boot(tg *target) (err error) {
	return amd64Linux.makeboot(tg.name, "MAKEINFO=missing make crossgcc-i386 && make "+tg.main)
}

func makeAmd64Linux(tg *target) error {
	return amd64Linux.godoforpkg(tg, "build", "-o", tg.name, tg.main)
}

func makeAmd64LinuxStatic(tg *target) error {
	return amd64Linux.godoforpkg(tg, "build", "-o", tg.name, "-tags", "netgo", tg.main)
}

func makeAmd64LinuxTest(tg *target) error {
	return amd64Linux.godoforpkg(tg, "test", "-c", "-o", tg.name, tg.main)
}

func makeAmd64CorebootRom(tg *target) (err error) {
	dir := "worktrees/coreboot/" + tg.main
	build := dir + "/build"
	cbfstool := build + "/cbfstool"
	tmprom := tg.name + ".tmp"

	cmdline := "cp " + build + "/coreboot.rom " + tmprom +
		" && " + cbfstool + " " + tmprom + " add-payload" +
		" -f " + tg.main + ".vmlinuz" +
		" -I goes-boot.cpio.xz" +
		` -C "console=ttyS1,57600n8 console=ttyS0,115200n8 intel_iommu=off quiet"` +
		" -n fallback/payload -c none -r COREBOOT" +
		" && mv " + tmprom + " " + tg.name +
		" && " + cbfstool + " " + tg.name + " print"
	if err := shellCommandRun(cmdline); err != nil {
		return err
	}
	return
}

func makeAmd64LinuxKernel(tg *target) (err error) {
	return amd64Linux.makeLinux(tg.name, tg.main)
}

func makeAmd64LinuxInitramfs(tg *target) (err error) {
	err = makeAmd64LinuxStatic(tg)
	if err != nil {
		return
	}
	return amd64Linux.makeCpioArchive(tg)
}

func makeHost(tg *target) error {
	return host.godoforpkg(tg, "build", "-o", tg.name, tg.main)
}

func makeHostTest(tg *target) error {
	return host.godoforpkg(tg, "test", "-c", "-o", tg.name, tg.main)
}

func makeGoesPlatinaMk1(tg *target) error {
	args := []string{}
	if strings.Index(*tagsFlag, "debug") >= 0 {
		args = append(args, "-gcflags", "-N -l")
	}
	return amd64Linux.godoforpkg(goesPlatinaMk1,
		append(append([]string{"build", "-o",
			goesPlatinaMk1.name},
			args...), goesPlatinaMk1.dir)...)
}

func makeGoesPlatinaMk1Installer(tg *target) error {
	var zfiles []string
	tinstaller := tg.name + ".tmp"
	tzip := goesPlatinaMk1.name + ".zip"
	err := makeGoesPlatinaMk1(tg)
	if err != nil {
		return err
	}
	err = amd64Linux.godoforpkg(tg, "build", "-o", tinstaller,
		platinaGoesMainGoesInstaller)
	if err != nil {
		return err
	}
	const fe1so = "fe1.so"
	fi, fierr := os.Stat(fe1so)
	if fierr != nil {
		return fmt.Errorf("can't find " + fe1so)
	}
	zfiles = append(zfiles, fi.Name())

	err = zipfile(tzip, append(zfiles, goesPlatinaMk1.name))
	if err != nil {
		return err
	}
	err = catto(tg.name, tinstaller, tzip)
	if err != nil {
		return err
	}
	if err = rm(tinstaller, tzip); err != nil {
		return err
	}
	if err = zipa(tg.name); err != nil {
		return err
	}
	return chmodx(tg.name)
}

func (goenv *goenv) makeCpioArchive(tg *target) (err error) {
	if *nFlag {
		return nil
	}
	arname := strings.TrimPrefix(tg.name+goenv.cpioSuffix,
		goenv.cpioTrimPrefix)
	f, err := os.Create(arname + ".tmp")
	if err != nil {
		return
	}
	defer func() {
		f.Close()
		if err == nil {
			mv(arname+".tmp", arname)
		} else {
			rm(arname + ".tmp")
		}
	}()
	rp, wp := io.Pipe()

	w := cpio.NewWriter(wp)

	cmd, err := filterCommand(rp, f, "xz", "--stdout", "--check=crc32", "-9")
	defer func() {
		errcmd := cmd.Wait()
		if err == nil {
			err = errcmd
		}
	}()
	defer func() {
		errclose := wp.Close()
		if err == nil {
			err = errclose
		}
	}()
	defer func() {
		errclose := w.Close()
		if err == nil {
			err = errclose
		}
	}()
	if err != nil {
		return err
	}
	for _, dir := range []struct {
		name string
		mode os.FileMode
	}{
		{".", 0775},
		{"boot", 0775},
		{"etc", 0775},
		{"etc/goes", 0775},
		{"etc/goes/sshd", 0700},
		{"etc/ssl", 0775},
		{"etc/ssl/certs", 0775},
		{"perm", 0775},
		{"sbin", 0775},
		{"usr", 0775},
		{"usr/bin", 0775},
		{"volatile", 0775},
	} {
		err = mkdirCpio(w, dir.name, dir.mode)
		if err != nil {
			return
		}
	}
	for _, file := range []struct {
		tname string
		mode  os.FileMode
		hname string
	}{
		{"etc/ssl/certs/ca-certificates.crt", 0644,
			"/etc/ssl/certs/ca-certificates.crt"},
		{"etc/goes/sshd/authorized_keys.default", 0600,
			platinaSecrets + "/secrets/sshd/id_rsa.pub"},
	} {
		if err = mkfileFromHostCpio(w, file.tname, file.mode, file.hname); err != nil {
			return
		}
	}

	if err = mkfileFromSliceCpio(w, "etc/resolv.conf", 0644, tg.name, []byte("nameserver 8.8.8.8\n")); err != nil {
		return
	}

	if err = mkfileFromSliceCpio(w, "etc/goes/init", 0644, tg.name, []byte("ip link lo change up\n")); err != nil {
		return
	}

	goesbin, err := goenv.stripBinary(tg.dir + "/" + tg.name)
	if err != nil {
		return
	}
	if err = mkfileFromSliceCpio(w, "init", 0755, "(stripped)"+tg.name, goesbin); err != nil {
		return
	}
	for _, link := range []struct {
		hname string
		tname string
	}{
		{"usr/bin/goes", "../../init"},
	} {
		if err = mklinkCpio(w, link.hname, link.tname); err != nil {
			return
		}
	}
	return
}

func mkdirCpio(w *cpio.Writer, name string, perm os.FileMode) (err error) {
	host.log("{archive}mkdir", "-m", fmt.Sprintf("%o", perm), name)
	hdr := &cpio.Header{
		Name: name,
		Mode: cpio.ModeDir | cpio.FileMode(perm),
	}
	err = w.WriteHeader(hdr)
	return
}

func mklinkCpio(w *cpio.Writer, name string, target string) (err error) {
	host.log("{archive}ln", "-s", name, target)
	link := []byte(target)
	hdr := &cpio.Header{
		Name: name,
		Mode: 0120777,
		Size: int64(len(link)),
	}
	if err = w.WriteHeader(hdr); err != nil {
		return
	}
	_, err = w.Write(link)
	return
}

func mkfileFromSliceCpio(w *cpio.Writer, tname string, mode os.FileMode, hname string, data []byte) (err error) {
	hdr := &cpio.Header{
		Name: tname,
		Mode: 0100000 | cpio.FileMode(mode),
		Size: int64(len(data)),
	}
	if err = w.WriteHeader(hdr); err != nil {
		return
	}
	if _, err = w.Write(data); err != nil {
		return
	}
	host.log("{archive}cp", hname, tname)
	return
}

func mkfileFromHostCpio(w *cpio.Writer, tname string, mode os.FileMode, hname string) (err error) {
	data, err := ioutil.ReadFile(hname)
	if err != nil {
		return
	}
	return mkfileFromSliceCpio(w, tname, mode, hname, data)
}

func (goenv *goenv) godoindir(dir string, args ...string) error {
	if len(*tagsFlag) > 0 {
		done := false
		for i, arg := range args {
			if arg == "-tags" {
				args[i+1] = fmt.Sprint(args[i+1], " ",
					*tagsFlag)
				done = true
			}
		}
		if !done {
			args = append([]string{args[0], "-tags", *tagsFlag},
				args[1:]...)
		}
	}
	if *nFlag {
		args = append([]string{args[0], "-n"}, args[1:]...)
	}
	if *vFlag {
		args = append([]string{args[0], "-v"}, args[1:]...)
	}
	if *xFlag {
		args = append([]string{args[0], "-x"}, args[1:]...)
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	if goenv.goarch != runtime.GOARCH {
		cmd.Env = append(cmd.Env, fmt.Sprint("GOARCH=", goenv.goarch))
	}
	if goenv.goos != runtime.GOOS {
		cmd.Env = append(cmd.Env, fmt.Sprint("GOOS", goenv.goos))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	goenv.log(cmd.Args...)
	return cmd.Run()
}

func (goenv *goenv) godo(args ...string) error {
	return goenv.godoindir(platinaGoes, args...)
}

func (goenv *goenv) godoforpkg(tg *target, args ...string) error {
	dir := tg.dir
	if dir == "" {
		dir = platinaGoes // legacy packages
	}
	return goenv.godoindir(dir, args...)
}

func (goenv *goenv) log(args ...string) {
	if !*zFlag {
		return
	}
	fmt.Print("#")
	if goenv.goarch != runtime.GOARCH || goenv.goos != runtime.GOOS {
		fmt.Print(" {", goenv.goarch, ",", goenv.goos, "}")
	}
	for _, arg := range args {
		format := " %s"
		if strings.ContainsAny(arg, " \t") {
			format = " %q"
		}
		fmt.Printf(format, arg)
	}
	fmt.Println()
}

func catto(target string, fns ...string) error {
	host.log(append(append([]string{"cat"}, fns...), ">>", target)...)
	w, err := os.Create(target)
	if err != nil {
		return err
	}
	defer w.Close()
	for _, fn := range fns {
		r, err := os.Open(fn)
		if err != nil {
			w.Close()
			return err
		}
		io.Copy(w, r)
		r.Close()
	}
	return nil
}

func chmodx(fn string) error {
	host.log("chmod", "+x", fn)
	fi, err := os.Stat(fn)
	if err != nil {
		return err
	}
	return os.Chmod(fn, fi.Mode()|
		os.FileMode(syscall.S_IXUSR|syscall.S_IXGRP|syscall.S_IXOTH))
}

func mv(from, to string) error {
	host.log("mv", from, to)
	return os.Rename(from, to)
}

func rm(fns ...string) error {
	host.log(append([]string{"rm"}, fns...)...)
	for _, fn := range fns {
		if err := os.Remove(fn); err != nil {
			return err
		}
	}
	return nil
}

// FIXME write a go method to prefix the self extractor header.
func zipa(fn string) error {
	cmd := exec.Command("zip", "-q", "-A", fn)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	host.log(cmd.Args...)
	if *nFlag {
		return nil
	}
	return cmd.Run()
}

func zipfile(zfn string, fns []string) error {
	host.log(append([]string{"zip", zfn}, fns...)...)
	f, err := os.Create(zfn)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	defer z.Close()
	for _, fn := range fns {
		w, err := z.Create(filepath.Base(fn))
		if err != nil {
			return err
		}
		r, err := os.Open(fn)
		if err != nil {
			return err
		}
		io.Copy(w, r)
		r.Close()
	}
	return nil
}

func filterCommand(in io.Reader, out io.Writer, name string, args ...string) (cmd *exec.Cmd, err error) {
	host.log(append([]string{name}, args...)...)
	if *nFlag {
		return nil, nil
	}
	cmd = exec.Command(name, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

func (goenv *goenv) stripBinary(in string) (out []byte, err error) {
	outfile := in + ".strip.tmp"
	cmdline := []string{"-o", outfile, in}
	stripper := goenv.gnuPrefix + "strip"
	host.log(append([]string{stripper}, cmdline...)...)
	if *nFlag {
		return nil, nil
	}
	defer rm(outfile)
	cmd := exec.Command(stripper, cmdline...)
	err = cmd.Run()
	if err != nil {
		return
	}
	out, err = ioutil.ReadFile(outfile)
	return
}

func shellCommand(cmdline string) (cmd *exec.Cmd) {
	args := []string{"-c", cmdline}
	if *xFlag {
		args = append(args, "-x")
	}
	host.log(append([]string{"sh"}, args...)...)
	if *nFlag {
		return nil
	}
	cmd = exec.Command("sh", args...)
	cmd.Env = os.Environ()
	return
}

func shellCommandOutput(cmdline string) (str string, err error) {
	cmd := shellCommand(cmdline)
	if cmd == nil {
		return
	}
	out, err := cmd.Output()
	if err != nil {
		return
	}
	str = strings.Trim(string(out), "\n")
	return
}

func shellCommandRun(cmdline string) (err error) {
	cmd := shellCommand(cmdline)
	if cmd == nil {
		return
	}
	if *zFlag {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return
}

func configWorktree(repo string, machine string, config string) (workdir string, err error) {
	var gitdir string
	for _, dir := range []string{
		filepath.Join(platina, repo),
		filepath.Join(platina, "src", repo),
		filepath.Join(platinaSystemBuildSrc, repo),
	} {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			var err error
			gitdir, err = filepath.Abs(dir)
			if err != nil {
				return "", fmt.Errorf("Can't make %s absolute: %s",
					dir, err)
			}
			break
		}
	}
	if len(gitdir) == 0 {
		return "", fmt.Errorf("can't find gitdir for %s", repo)
	}
	workdir = filepath.Join("worktrees", repo, machine)
	_, err = os.Stat(workdir)
	if os.IsNotExist(err) {
		clone := ""
		if *cloneFlag {
			clone = " || git clone . $p"
		}
		if err := shellCommandRun("mkdir -p " + workdir +
			" && cd " + workdir +
			" && p=`pwd` " +
			" && b=worktree_`pwd | sed -e 's,/,_,g'`" +
			" && cd " + gitdir +
			" && ( git worktree prune ; git branch -d $b" +
			" ; git worktree add -b $b $p" +
			clone +
			" )" +
			" && cd $p" +
			" && " + config); err != nil {
			return "", err
		}
		err = nil
		return
	}
	if err == nil {
		if *rFlag != "" {
			if err := shellCommandRun("cd " + workdir +
				" && git rebase " + *rFlag +
				" && " + config); err != nil {
				return "", err
			}
		}
	}
	return
}

func (goenv *goenv) makeboot(out string, configCommand string) (err error) {
	machine := strings.TrimPrefix(out, goenv.boot+"-")
	dir, err := configWorktree(goenv.boot, machine, configCommand)
	if err != nil {
		return
	}
	cmdline := "make -C " + dir +
		" ARCH=" + goenv.kernelArch +
		" CROSS_COMPILE=" + goenv.gnuPrefix
	if !*zFlag { // quiet "Skipping submodule and Created CBFS" messages
		cmdline += " 2>/dev/null"
	}
	if err := shellCommandRun(cmdline); err != nil {
		return err
	}
	return
}

func (goenv *goenv) makeLinux(out string, config string) (err error) {
	machine := strings.TrimSuffix(out, ".vmlinuz")
	configCommand := "cp " + goenv.kernelConfigPath + "/" + config +
		" .config" +
		" && make oldconfig ARCH=" + goenv.kernelArch

	dir, err := configWorktree("linux", machine, configCommand)
	if err != nil {
		return
	}
	ver, err := shellCommandOutput("cd " + dir + " && git describe")
	if err != nil {
		return err
	}
	ver = strings.TrimLeft(ver, "v")
	f := strings.Split(ver, "-")
	id := f[0] + "-" + machine
	if err := shellCommandRun("make -C " + dir +
		" -j " + strconv.Itoa(runtime.NumCPU()*2) +
		" ARCH=" + goenv.kernelArch +
		" CROSS_COMPILE=" + goenv.gnuPrefix +
		" KDEB_PKGVERSION=" + ver +
		" KERNELRELEASE=" + id + " " +
		goenv.kernelMakeTarget); err != nil {
		return err
	}
	cmdline := "cp " + dir + "/" + goenv.kernelPath + " " + out
	if err := shellCommandRun(cmdline); err != nil {
		return err
	}
	return
}
