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
	"strings"
	"syscall"

	"github.com/cavaliercoder/go-cpio"
)

const (
	platina            = ".."
	platinaFe1         = platina + "/fe1"
	platinaFe1Firmware = platina + "/firmware-fe1a"
	platinaGo          = platina + "/go"
	platinaGoMain      = platinaGo + "/main"

	platinaVnetMk1 = platina + "/vnet-platina-mk1"

	platinaSystemBuildSrc = platina + "/system-build/src"

	platinaGoMainIP                   = platinaGoMain + "/ip"
	platinaGoMainGoesPrefix           = platinaGoMain + "goes-"
	platinaGoMainGoesExample          = platina + "/goes-example"
	platinaGoMainGoesBoot             = platina + "/goes-boot"
	platinaGoMainGoesInstaller        = platinaGoMain + "/goes-installer"
	platinaGoMainGoesPlatinaMk1       = platina + "/goes-platina-mk1"
	platinaGoMainGoesPlatinaMk1Bmc    = platina + "/goes-bmc"
	platinaGoMainGoesPlatinaMk2       = platinaGoMain + "/goes-platina-mk2"
	platinaGoMainGoesPlatinaMk2Lc1Bmc = platinaGoMainGoesPlatinaMk2 + "-lc1-bmc"
	platinaGoMainGoesPlatinaMk2Mc1Bmc = platinaGoMainGoesPlatinaMk2 + "-mc1-bmc"

	goesExample             = "goes-example"
	goesExampleArm          = "goes-example-arm"
	goesBoot                = "goes-boot"
	goesBootArm             = "goes-boot-arm"
	goesIP                  = "goes-ip"
	goesIPTest              = "goes-ip.test"
	goesPlatinaMk1          = "goes-platina-mk1"
	goesPlatinaMk1Installer = "goes-platina-mk1-installer"
	goesPlatinaMk1Test      = "goes-platina-mk1.test"
	goesPlatinaMk1Bmc       = "goes-platina-mk1-bmc"
	goesPlatinaMk2Lc1Bmc    = "goes-platina-mk2-lc1-bmc"
	goesPlatinaMk2Mc1Bmc    = "goes-platina-mk2-mc1-bmc"

	corebootExampleAmd64        = "coreboot-example-amd64"
	corebootExampleAmd64Config  = "example-amd64_defconfig"
	corebootExampleAmd64Machine = "example-amd64"
	corebootExampleAmd64Rom     = "coreboot-example-amd64.rom"
	corebootPlatinaMk1          = "coreboot-platina-mk1"
	corebootPlatinaMk1Config    = "platina-mk1_defconfig"
	corebootPlatinaMk1Machine   = "platina-mk1"
	corebootPlatinaMk1Rom       = "coreboot-platina-mk1.rom"

	exampleAmd64Vmlinuz     = "example-amd64.vmlinuz"
	platinaMk1Vmlinuz       = "platina-mk1.vmlinuz"
	platinaMk1BmcVmlinuz    = "platina-mk1-bmc.vmlinuz"
	platinaMk2Lc1BmcVmlinuz = "platina-mk2-lc1-bmc.vmlinuz"
	platinaMk2Mc1BmcVmlinuz = "platina-mk2-mc1-bmc.vmlinuz"

	ubootPlatinaMk1Bmc = "u-boot-platina-mk1-bmc"

	vnetPlatinaMk1 = "vnet-platina-mk1"

	zipPlatinaMk1Bmc = "platina-mk1-bmc.zip"
)

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
	defaultTargets = []string{
		goesExample,
		exampleAmd64Vmlinuz,
		corebootExampleAmd64,
		goesExampleArm,
		goesBoot,
		goesBootArm,
		goesIP,
		goesPlatinaMk1,
		vnetPlatinaMk1,
		platinaMk1Vmlinuz,
		corebootPlatinaMk1,
		goesPlatinaMk1Bmc,
		platinaMk1BmcVmlinuz,
		ubootPlatinaMk1Bmc,
		platinaMk2Lc1BmcVmlinuz,
		platinaMk2Mc1BmcVmlinuz,
		corebootExampleAmd64Rom,
		corebootPlatinaMk1Rom,
		zipPlatinaMk1Bmc,
	}
	goarchFlag = flag.String("goarch", runtime.GOARCH,
		"GOARCH of PACKAGE build")
	goosFlag = flag.String("goos", runtime.GOOS,
		"GOOS of PACKAGE build")
	cloneFlag = flag.Bool("clone", false,
		"Fallback to 'git clone' if git worktree does not work.")
	nFlag = flag.Bool("n", false,
		"print 'go build' commands but do not run them.")
	oFlag    = flag.String("o", "", "output file name of PACKAGE build")
	rFlag    = flag.Bool("r", false, "rebase worktrees before build")
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
		kernelMakeTarget: "bzImage bindeb-pkg",
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
	mainPkg = map[string]string{
		goesExample:             platinaGoMainGoesExample,
		exampleAmd64Vmlinuz:     "platina-example-amd64_defconfig",
		corebootExampleAmd64:    corebootExampleAmd64Config,
		corebootExampleAmd64Rom: corebootExampleAmd64Machine,
		goesExampleArm:          platinaGoMainGoesExample,
		goesBoot:                platinaGoMainGoesBoot,
		goesBootArm:             platinaGoMainGoesBoot,
		goesIP:                  platinaGoMainIP,
		goesIPTest:              platinaGoMainIP,
		goesPlatinaMk1:          platinaGoMainGoesPlatinaMk1,
		vnetPlatinaMk1:          platinaVnetMk1,
		platinaMk1Vmlinuz:       "platina-mk1_defconfig",
		corebootPlatinaMk1:      corebootPlatinaMk1Config,
		corebootPlatinaMk1Rom:   corebootPlatinaMk1Machine,
		goesPlatinaMk1Test:      platinaGoMainGoesPlatinaMk1,
		goesPlatinaMk1Installer: platinaGoMainGoesPlatinaMk1,
		goesPlatinaMk1Bmc:       platinaGoMainGoesPlatinaMk1Bmc,
		platinaMk1BmcVmlinuz:    "platina-mk1-bmc_defconfig",
		ubootPlatinaMk1Bmc:      "platinamx6boards_qspi_defconfig",
		goesPlatinaMk2Lc1Bmc:    platinaGoMainGoesPlatinaMk2Lc1Bmc,
		platinaMk2Lc1BmcVmlinuz: "platina-mk2-lc1-bmc_defconfig",
		goesPlatinaMk2Mc1Bmc:    platinaGoMainGoesPlatinaMk2Mc1Bmc,
		platinaMk2Mc1BmcVmlinuz: "platina-mk2-mc1-bmc_defconfig",
	}
	makeFun map[string]func(out, name string) error
	pkgdir  = map[string]string{
		goesBoot:          "../goes-boot",
		goesBootArm:       "../goes-boot",
		goesPlatinaMk1:    "../goes-platina-mk1",
		goesPlatinaMk1Bmc: "../goes-bmc",
		goesExample:       "../goes-example",
		goesExampleArm:    "../goes-example",
		vnetPlatinaMk1:    "../vnet-platina-mk1",
	}
	pkgbuilt = map[string]bool{}
)

func init() {
	flag.Usage = usage
	makeFun = map[string]func(out, name string) error{
		goesExample:             makeHost,
		exampleAmd64Vmlinuz:     makeAmd64LinuxKernel,
		corebootExampleAmd64:    makeAmd64Boot,
		corebootExampleAmd64Rom: makeAmd64CorebootRom,
		goesExampleArm:          makeArmLinuxStatic,
		goesBoot:                makeAmd64LinuxInitramfs,
		goesBootArm:             makeArmLinuxInitramfs,
		goesIP:                  makeHost,
		goesIPTest:              makeHostTest,
		goesPlatinaMk1:          makeGoesPlatinaMk1,
		vnetPlatinaMk1:          makeAmd64LinuxStatic,
		platinaMk1Vmlinuz:       makeAmd64LinuxKernel,
		corebootPlatinaMk1:      makeAmd64Boot,
		corebootPlatinaMk1Rom:   makeAmd64CorebootRom,
		goesPlatinaMk1Installer: makeGoesPlatinaMk1Installer,
		goesPlatinaMk1Test:      makeAmd64LinuxTest,
		goesPlatinaMk1Bmc:       makeArmLinuxInitramfs,
		platinaMk1BmcVmlinuz:    makeArmLinuxKernel,
		ubootPlatinaMk1Bmc:      makeArmBoot,
		goesPlatinaMk2Lc1Bmc:    makeArmLinuxStatic,
		platinaMk2Lc1BmcVmlinuz: makeArmLinuxKernel,
		goesPlatinaMk2Mc1Bmc:    makeArmLinuxStatic,
		platinaMk2Mc1BmcVmlinuz: makeArmLinuxKernel,
		zipPlatinaMk1Bmc:        makeArmZipfile,
	}
}

func makeDependent(target string) {
	if pkgbuilt[target] {
		fmt.Printf("# Dependent package %s already built\n", target)
		return
	}
	fmt.Printf("# Making dependent package %s\n", target)
	makeTarget(target)
}

func makeTarget(target string) {
	var err error
	if pkgbuilt[target] {
		fmt.Printf("# Package %s already built\n", target)
		return
	}
	if f, found := makeFun[target]; found {
		err = f(target, mainPkg[target])
	} else {
		err = makePackage(target)
	}
	if err != nil {
		fmt.Printf("Error making package %s\n", target)
		panic(err)
	}
	pkgbuilt[target] = true
}

func main() {
	flag.Parse()
	targets := flag.Args()
	if len(targets) == 0 {
		targets = defaultTargets
	} else if targets[0] == "all" {
		targets = targets[:0]
		for target := range makeFun {
			targets = append(targets, target)
		}
	}
	for _, target := range targets {
		makeTarget(target)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:", os.Args[0],
		"[ OPTION... ] [ TARGET... | PACKAGE ]")
	fmt.Fprintln(os.Stderr, "\nOptions:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nDefault Targets:")
	for _, target := range defaultTargets {
		fmt.Fprint(os.Stderr, "\t", target, "\n")
	}
	fmt.Fprintln(os.Stderr, "\n\"all\" Targets:")
	for target := range makeFun {
		fmt.Fprint(os.Stderr, "\t", target, "\n")
	}
}

func makeArmLinuxStatic(out, name string) error {
	return armLinux.godoforpkg(out, "build", "-o", out, "-tags", "netgo",
		"-ldflags", "-d", name)
}

func makeArmBoot(out, name string) (err error) {
	machine := strings.TrimPrefix(out, "u-boot-")
	makeDependent(machine + ".vmlinuz")
	cmdline := "cp " + machine + ".vmlinuz " + machine + "-ker.bin"
	if err := shellCommandRun(cmdline); err != nil {
		return err
	}
	if err = armLinux.makeboot(out, "make "+name); err != nil {
		return err
	}
	env := makeUbootEnv()
	if err = ioutil.WriteFile(machine+"-env.bin", env, 0644); err != nil {
		return err
	}
	cmdline = "cp worktrees/linux/" + machine + "/arch/arm/boot/dts/" +
		machine + ".dtb " + machine + "-dtb.bin"
	if err := shellCommandRun(cmdline); err != nil {
		return err
	}

	uboot := makeUboot("worktrees/u-boot/" + machine + "/u-boot.imx")
	if err = ioutil.WriteFile(machine+"-ubo.bin", uboot, 0644); err != nil {
		return err
	}

	return nil
}

func makeArmZipfile(out, name string) (err error) {
	machine := strings.TrimSuffix(out, ".zip")

	makeDependent("u-boot-" + machine)

	makeDependent(goesPlatinaMk1Bmc)
	cmdline := "mkimage -C none -A arm -O linux -T ramdisk -d " +
		machine + ".cpio.xz " + machine + "-ini.bin"
	if err := shellCommandRun(cmdline); err != nil {
		return err
	}

	makeVer("rel") // FIXME

	zipFile, err := os.Create(machine + ".zip")
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, suffix := range []string{
		"-env.bin",
		"-ker.bin",
		"-ini.bin",
		"-ubo.bin",
		"-ver.bin",
		"-dtb.bin",
	} {
		file, err := os.Open(machine + suffix)
		if err != nil {
			fmt.Printf("Error opening %s: %s\n", machine+suffix,
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

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}

		// Change to deflate to gain better compression
		// see http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}
		if _, err = io.Copy(writer, file); err != nil {
			os.Remove(machine + ".zip")
			panic(err)
		}
		armLinux.log("added", machine+suffix, "to", machine+".zip")
	}
	return nil
}

func makeArmLinuxKernel(out, name string) (err error) {
	return armLinux.makeLinux(out, name)
}

func makeArmLinuxInitramfs(out, name string) (err error) {
	err = makeArmLinuxStatic(out, name)
	if err != nil {
		return
	}
	return armLinux.makeCpioArchive(out)
}

func makeAmd64Boot(out, name string) (err error) {
	return amd64Linux.makeboot(out, "make crossgcc-i386 && make "+name)
}

func makeAmd64Linux(out, name string) error {
	return amd64Linux.godoforpkg(out, "build", "-o", out, name)
}

func makeAmd64LinuxStatic(out, name string) error {
	return amd64Linux.godoforpkg(out, "build", "-o", out, "-tags", "netgo", name)
}

func makeAmd64LinuxTest(out, name string) error {
	return amd64Linux.godoforpkg(out, "test", "-c", "-o", out, name)
}

func makeAmd64CorebootRom(romfile, machine string) (err error) {
	makeDependent("coreboot-" + machine)
	makeDependent(machine + ".vmlinuz")
	makeDependent(goesBoot)

	dir := "worktrees/coreboot/" + machine
	build := dir + "/build"
	cbfstool := build + "/cbfstool"
	tmprom := romfile + ".tmp"

	cmdline := "cp " + build + "/coreboot.rom " + tmprom +
		" && " + cbfstool + " " + tmprom + " add-payload" +
		" -f " + machine + ".vmlinuz" +
		" -I goes-boot.cpio.xz" +
		" -C console=ttyS0" +
		" -n fallback/payload -t payload -c none -r COREBOOT" +
		" && mv " + tmprom + " " + romfile +
		" && " + cbfstool + " " + romfile + " print"
	if err := shellCommandRun(cmdline); err != nil {
		return err
	}
	return
}

func makeAmd64LinuxKernel(out, name string) (err error) {
	return amd64Linux.makeLinux(out, name)
}

func makeAmd64LinuxInitramfs(out, name string) (err error) {
	err = makeAmd64LinuxStatic(out, name)
	if err != nil {
		return
	}
	return amd64Linux.makeCpioArchive(out)
}

func makeHost(out, name string) error {
	return host.godoforpkg(out, "build", "-o", out, name)
}

func makeHostTest(out, name string) error {
	return host.godoforpkg(out, "test", "-c", "-o", out, name)
}

func makePackage(name string) error {
	args := []string{"build"}
	if len(*oFlag) > 0 {
		args = append(args, "-o", *oFlag)
	}
	return (&goenv{goarch: *goarchFlag, goos: *goosFlag}).godo(append(args, name)...)
}

func makeGoesPlatinaMk1(out, name string) error {
	args := []string{}
	if strings.Index(*tagsFlag, "debug") >= 0 {
		args = append(args, "-gcflags", "-N -l")
	}
	return amd64Linux.godoforpkg(out, append(append([]string{"build", "-o", out}, args...), name)...)
}

func makeGoesPlatinaMk1Installer(out, name string) error {
	var zfiles []string
	tinstaller := out + ".tmp"
	tzip := goesPlatinaMk1 + ".zip"
	err := makeGoesPlatinaMk1(goesPlatinaMk1, name)
	if err != nil {
		return err
	}
	err = amd64Linux.godoforpkg(out, "build", "-o", tinstaller,
		platinaGoMainGoesInstaller)
	if err != nil {
		return err
	}
	const fe1so = "fe1.so"
	fi, fierr := os.Stat(fe1so)
	if fierr != nil {
		return fmt.Errorf("can't find " + fe1so)
	}
	zfiles = append(zfiles, fi.Name())

	err = zipfile(tzip, append(zfiles, goesPlatinaMk1))
	if err != nil {
		return err
	}
	err = catto(out, tinstaller, tzip)
	if err != nil {
		return err
	}
	if err = rm(tinstaller, tzip); err != nil {
		return err
	}
	if err = zipa(out); err != nil {
		return err
	}
	return chmodx(out)
}

func (goenv *goenv) makeCpioArchive(name string) (err error) {
	if *nFlag {
		return nil
	}
	arname := strings.TrimPrefix(name+goenv.cpioSuffix,
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
		{"etc", 0775},
		{"etc/ssl", 0775},
		{"etc/ssl/certs", 0775},
		{"sbin", 0775},
		{"usr", 0775},
		{"usr/bin", 0775},
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
		{"etc/ssl/certs/ca-certificates.crt", 0644, "/etc/ssl/certs/ca-certificates.crt"},
	} {
		if err = mkfileFromHostCpio(w, file.tname, file.mode, file.hname); err != nil {
			return
		}
	}

	if err = mkfileFromSliceCpio(w, "/etc/resolv.conf", 0644, name, []byte("nameserver 8.8.8.8\n")); err != nil {
		return
	}

	goesbin, err := goenv.stripBinary(pkgdir[name] + "/" + name)
	if err != nil {
		return
	}
	if err = mkfileFromSliceCpio(w, "/usr/bin/goes", 0755, "(stripped)"+name, goesbin); err != nil {
		return
	}
	for _, link := range []struct {
		hname string
		tname string
	}{
		{"init", "/usr/bin/goes"},
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
	return goenv.godoindir("../go", args...)
}

func (goenv *goenv) godoforpkg(pkg string, args ...string) error {
	dir := pkgdir[pkg]
	if dir == "" {
		dir = "../go" // legacy packages
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
		if *rFlag {
			if err := shellCommandRun("cd " + workdir +
				" && git fetch origin " +
				" && git rebase origin" +
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
