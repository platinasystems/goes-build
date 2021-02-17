CP=/bin/cp
INSTALL=/usr/bin/install
COREBOOTBIN=worktrees/coreboot/

goes-build:
	go build
	#./goes-build -x -z -v coreboot-example-amd64 coreboot-platina-mk1

install:
	#$(INSTALL) -d $(DESTDIR)/usr/bin
	#$(INSTALL) goes-build $(DESTDIR)/usr/bin
	#$(INSTALL) -d $(DESTDIR)/usr/share/goes-build/binary
	#$(CP) $(COREBOOTBIN)/example-amd64/build/coreboot.rom $(DESTDIR)/usr/share/goes-build/binary/coreboot-example-amd64.rom
	#$(CP) $(COREBOOTBIN)/platina-mk1/build/coreboot.rom $(DESTDIR)/usr/share/goes-build/binary/coreboot-platina-mk1.rom

clean:
	rm -f *.rom debian/debhelper-build-stamp debian/files debian/*.substvars *.vmlinuz *.xz *.bin *.zip goes-build
	rm -rf debian/.debhelper debian/goes-build

bindeb-pkg:
	debuild -i -us -uc -I -Iworktrees --lintian-opts --profile debian

.PHONY: goes-build install clean binpkg-deb
