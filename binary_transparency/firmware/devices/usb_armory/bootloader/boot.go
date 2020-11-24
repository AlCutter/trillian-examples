// https://github.com/f-secure-foundry/armory-boot
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//+build armory

package main

import (
	"bytes"
	"debug/elf"
	"fmt"

	"github.com/f-secure-foundry/tamago/arm"
	"github.com/f-secure-foundry/tamago/dma"
	"github.com/f-secure-foundry/tamago/soc/imx6"

	usbarmory "github.com/f-secure-foundry/tamago/board/f-secure/usbarmory/mark-two"
)

// defined in boot.s
func exec(kernel uint32, params uint32)
func execElf(elf uint32)
func svc()

func bootElf(img []byte) {
	dma.Init(dmaStart, dmaSize)
	mem, _ := dma.Reserve(dmaSize, 0)

	f, err := elf.NewFile(bytes.NewReader(img))
	if err != nil {
		panic(err.Error)
	}

	for idx, prg := range f.Progs {
		if prg.Type == elf.PT_LOAD {
			fmt.Printf("Loading %+v\n", *prg)
			b := make([]byte, prg.Memsz)
			n, err := prg.ReadAt(b[0:prg.Filesz], 0)
			if err != nil {
				panic(fmt.Sprintf("Failed to read LOAD section at idx %d: %q", idx, err))
			}
			dma.Write(mem, b, int(prg.Paddr))
			fmt.Printf("Loaded %d bytes at 0x%x\n", n, prg.Paddr)
		} else {
			fmt.Printf("ignoring %+v\n", *prg)
		}
	}

	entry := f.Entry

	arm.ExceptionHandler(func(n int) {
		if n != arm.SUPERVISOR {
			panic("unhandled exception")
		}

		fmt.Printf("armory-boot: starting elf image@%x\n", entry)

		usbarmory.LED("blue", false)
		usbarmory.LED("white", false)
		execElf(uint32(entry))
	})

	svc()
}

func boot(kernel []byte, dtb []byte, cmdline string) {
	dma.Init(dmaStart, dmaSize)
	mem, _ := dma.Reserve(dmaSize, 0)

	dma.Write(mem, kernel, kernelOffset)
	dma.Write(mem, dtb, dtbOffset)

	image := mem + kernelOffset
	params := mem + dtbOffset

	arm.ExceptionHandler(func(n int) {
		if n != arm.SUPERVISOR {
			panic("unhandled exception")
		}

		fmt.Printf("armory-boot: starting kernel image@%x params@%x\n", image, params)

		usbarmory.LED("blue", false)
		usbarmory.LED("white", false)

		// Linux RNGB driver doesn't play well with a previously
		// initialized RNGB, therefore reset it.
		imx6.RNGB.Reset()

		imx6.ARM.InterruptsDisable()
		imx6.ARM.CacheFlushData()
		imx6.ARM.CacheDisable()

		exec(image, params)
	})

	svc()
}
