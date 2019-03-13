package main

import (
	"encoding/binary"
	"hash/crc32"
	"strings"
)

const envvar = `baudrate=115200
bootargs=console=ttymxc0,115200n8 mem=1024m init=/init start ip=dhcp
bootcmd=run readmac sfboot
bootdelay=3
boot_linux=bootz ${loadaddr} ${initrd_addr} ${fdt_addr}
dlbmc=mw.b 80800000 00 00600000;run dw_hdr;run dw_uboot;run dw_fdt;run dw_kernel;run dw_initrd
dw_fdt=tftpboot 80880000 ${serverip}:platina-mk1-bmc.dtb;setenv sz_fdt ${filesize}
dw_hdr=tftpboot 80800400 ${serverip}:qspi-header-sckl00;setenv sz_hdr ${filesize}
dw_initrd=tftpboot 80B00000 ${serverip}:initrd.img.xz;setenv sz_initrd ${filesize}
dw_kernel=tftpboot 80900000 ${serverip}:zImage;setenv sz_kernel ${filesize}
dw_uboot=tftpboot 80801000 ${serverip}:u-boot.imx;setenv sz_uboot ${filesize}
ethact=FEC
ethprime=FEC
fdt_addr=0x88000000
fdt_high=0xffffffff
fileaddr=80b00000
filesize=1fa758
gatewayip=192.168.101.1
initrd_addr=0x89000000
initrd_high=0xffffffff
ipaddr=192.168.101.100
loadaddr=0x82000000
load_fdt_net=${netmethod} ${fdt_addr} ${netserver}platina-mk1-bmc-dtb.bin
load_fdt_sf=sf read ${fdt_addr} 0x00080000 ${sz_fdt}
load_initramfs_net=${netmethod} ${initrd_addr} $(netserver}platina-mk1-bmc-ini.bin
load_initramfs_sf=sf read ${initrd_addr} 0x00300000 ${sz_initrd}
load_kernel_net=${netmethod} ${loadaddr} ${netserver}platina-mk1-bmc-ker.bin
load_kernel_sf=sf read ${loadaddr} 0x00100000 ${sz_kernel}
mask=255.255.255.0
netboot=run readmac load_kernel_net load_initramfs_net load_fdt_net boot_linux
netmask=255.255.255.0
netmethod=dhcp
qspi=sf probe;sf erase 0 00600000;sf erase fc0000 40000;sf write 80800000 0 00600000;saveenv
qspi0=mw 020e01b8 00000005; mw 20a8004 c7000000; mw 020a8000 4300ca05
qspi1=mw 020e01b8 00000005; mw 20a8004 c7000000; mw 020a8000 c300ca05
readmac=i2c read 55 0.2 200 80800000; setmac 80800000 24; saveenv
serverip=192.168.101.1
sfboot=sf probe 0;run load_kernel_sf load_initramfs_sf load_fdt_sf boot_linux
stderr=serial
stdin=serial
stdout=serial
sz_env=4000
sz_fdt=f000
sz_hdr=200
sz_initrd=300000
sz_kernel=200000
sz_uboot=5b718
wd=mw 020e01a0 00000005;mw 020e01a4 00000005;mw 020e01a8 00000005;mw 020e01b8 00000005;mw 020e01bc 00000005;mw 020a8000 0300ca05;mw 020a8004 07000000
`

const ubootEnvsize = 8192

func makeUbootEnv() []byte {
	binenv := make([]byte, ubootEnvsize)
	copy(binenv[crc32.Size:], strings.Replace(envvar, "\n", "\x00", -1))
	crc := crc32.ChecksumIEEE(binenv[crc32.Size:])
	binary.LittleEndian.PutUint32(binenv[:crc32.Size], crc)
	return binenv
}
