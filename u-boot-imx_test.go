package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestBMCQSPIConfig(t *testing.T) {
	file, err := os.Open("testdata/qspi-header-sckl00")
	if err != nil {
		t.Error("open testdata/qspi-header-sclk00 failed:", err)
		return
	}
	defer file.Close()

	td := QuadSPIConfig{}
	err = binary.Read(file, binary.LittleEndian, &td)
	if err != nil {
		t.Error("reading testdata:", err)
		return
	}

	imx := bmcQuadSPIConfig()

	if !reflect.DeepEqual(td, imx) {
		fmt.Println("Expected:")
		fmt.Println(td)
		fmt.Println("")
		fmt.Println("Got:")
		fmt.Println(imx)
		t.Error("data mismatch")
	}
}
