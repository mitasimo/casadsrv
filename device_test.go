package main

import (
	"testing"
	"time"

	"github.com/tarm/serial"
)

var (
	enq      = []byte{0x05}
	dc1      = []byte{0x11}
	asc byte = 0x06
)

func TestSerialPort(t *testing.T) {

	conn, err := serial.OpenPort(
		&serial.Config{
			Name:        "com3",
			Baud:        9600,
			ReadTimeout: time.Second,
		})
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer conn.Close()

	// for i := 0; i < 4; i++ {
	conn.Flush()

	nBytes, err := conn.Write(enq)
	if err != nil {
		t.Fatalf("Error write enq command: %v", err)
	}
	if nBytes != len(enq) {
		t.Fatalf("Error write enq command: try to send %d bytes, sent %d bytes", len(enq), nBytes)
	}

	ascBuff := make([]byte, 1)

	nBytes, err = conn.Read(ascBuff)
	if err != nil {
		t.Fatalf("Error read ASC: %v", err)
	}
	if nBytes != 1 {
		t.Fatalf("Error read ASC. Length of ASC is %d byte, is readed %d bytes", len(ascBuff), nBytes)
	}
	if ascBuff[0] != asc {
		t.Fatalf("readed %X, but ASC = %X", ascBuff[0], asc)
	}
	// conn.Flush()

	nBytes, err = conn.Write(dc1)
	if err != nil {
		t.Fatalf("Error sent DC1 command: %v", err)
	}
	if nBytes != len(dc1) {
		t.Fatalf("DC1 command len = %d, but sent %d bytes", len(dc1), nBytes)
	}

	rBuff := make([]byte, 1)

	// поиск заголовка
	cnt := 0
	for ; cnt < 25; cnt++ {
		nBytes, err = conn.Read(rBuff)
		if err != nil {
			t.Fatalf("Error read data: %v", err)
		}
		if rBuff[0] == 0x02 {
			break
		}
	}
	if cnt == 25 {
		t.Fatalf("Fire in the hole")
	}

	buff := make([]byte, 0, 1024)
	for {
		nBytes, err = conn.Read(rBuff)
		if err != nil {
			t.Fatalf("Error read data: %v", err)
		}
		if rBuff[0] == 0x03 {
			break
		}
		buff = append(buff, rBuff[0])
	}

	// if i == 3 {
	t.Fatalf("Stable=%s Sign=%s Weigth=%s %s", buff[0:1], buff[1:2], buff[2:8], buff[8:10])
	// 	}
	// }

}
