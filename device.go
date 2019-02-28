package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// NewSerialCasadDevice create new SerialCasadDevice
func NewSerialCasadDevice(portName string, portSpeed int) (*SerialCasadDevice, error) {

	// создать канал для отправки запросов
	qchan := make(chan chan weigthResult)

	// запустить горутину для обмена с устройством
	go func() {
		var (
			weigth    float64
			unstable  bool
			err       error
			timeStamp = time.Now().Add(time.Second * (-5)) // время последнего обмена с устройством
		)
		for rchan := range qchan {
			if err != nil || time.Since(timeStamp) > time.Second {
				// предыдущий обмен был либо с ошибкой, либо более секунды назад
				// необходимо выполнить обмен с устройством
				weigth, unstable, err = getWeigth(portName, portSpeed)
				timeStamp = time.Now()
			}
			rchan <- weigthResult{weigth, unstable, err}
			close(rchan)
		}
	}()

	return &SerialCasadDevice{queryChan: qchan}, nil
}

// тип значения в канале результата
type weigthResult struct {
	Weigth   float64
	Unstable bool
	Error    error
}

// SerialCasadDevice -
type SerialCasadDevice struct {
	queryChan chan chan weigthResult
}

// GetWeigth -
func (d *SerialCasadDevice) GetWeigth() (weigth float64, unstable bool, err error) {

	// канал для получения результата
	resultChan := make(chan weigthResult, 1)
	// запрос на получение веса (данных из устройства)
	d.queryChan <- resultChan
	// получить вес
	res, ok := <-resultChan
	if !ok {
		err = errors.New("Ошибка: канал устройства закрыт")
	} else {
		weigth = res.Weigth
		unstable = res.Unstable
		err = res.Error
	}
	return
}

// Disconnect -
func (d *SerialCasadDevice) Disconnect() {
	// закрыть канал запросов
	close(d.queryChan)
}

// getWeigth открывает последовательный порт,
// отправляет запрос устройству,
// получает и разбирает ответ
func getWeigth(portName string, portSpeed int) (weigth float64, unstable bool, err error) {
	// открыть последовательный порт
	cfg := &serial.Config{
		Name:        portName,
		Baud:        portSpeed,
		ReadTimeout: time.Second,
	}
	conn, err := serial.OpenPort(cfg)
	if err != nil {
		err = fmt.Errorf("Ошибка открытия последовательного порта %s: %v", portName, err)
		return
	}
	defer conn.Close()

	// очистить все буферы, прекратить людые операции ввода/вывода с портом
	conn.Flush()

	// отправить команду ENQ
	enq := []byte{0x05}
	nBytes, err := conn.Write(enq)
	if err != nil {
		err = fmt.Errorf("Ошибка отправки команды ENQ: %v", err)
		return
	}
	if nBytes != len(enq) {
		err = fmt.Errorf("Ошибка отправления команды ENQ: требуется отправить %d байт, было отправлено %d байт", len(enq), nBytes)
		return
	}

	// получить подтверждение ASC
	const ASC = 0x06
	ascBuff := make([]byte, 1)
	nBytes, err = conn.Read(ascBuff)
	if err != nil {
		err = fmt.Errorf("Ошибка получения подтверждения ASC: %v", err)
		return
	}
	if nBytes != 1 {
		err = fmt.Errorf("Ошибка получения подтверждения ASC. Длина подтверждения ASC %d байт. Получено %d байт", len(ascBuff), nBytes)
		return
	}
	if ascBuff[0] != ASC {
		err = fmt.Errorf("Ошибка получения подтверждения ASC. Получено значение %X. Подтверждение ASC должно быть %X", ascBuff, []byte{ASC})
		return
	}

	// отправить команду DC1
	dc1 := []byte{0x11}
	nBytes, err = conn.Write(dc1)
	if err != nil {
		err = fmt.Errorf("Ошибка отправки комманды DC1: %v", err)
		return
	}
	if nBytes != len(dc1) {
		err = fmt.Errorf("Ошибка отправки комманды DC1: длина команды %d байт, отправлено %d байт", len(dc1), nBytes)
		return
	}

	rBuff := make([]byte, 1) // буфер для чтения

	// параметры ответа
	const (
		numPred    = 25   // читать не более numPred байт для до признака начало ответа
		numBody    = 25   // читать не более numBody байт для до признака конца ответа
		sSign      = 0x02 // признак начало ответа
		eSign      = 0x03 // признак конца ответа
		stabSign   = 0x53 // признак стабильного показания индикатора (символ S).
		unstabSign = 0x55 // Нестабильный - симмол  U (код 0x55)
	)

	// поиск заголовка ответа
	num := 0
	for ; num < numPred; num++ {
		nBytes, err = conn.Read(rBuff)
		if err != nil {
			err = fmt.Errorf("Ошибка получения ответа: %v", err)
			return
		}
		if rBuff[0] == sSign {
			break
		}
	}
	if num >= numPred {
		err = fmt.Errorf("Ошибка получения ответа: превышено количество символов до начала ответа")
		return
	}

	// чтение тела ответа
	buff := make([]byte, 0, 1024)
	num = 0
	for ; num < numBody; num++ {
		nBytes, err = conn.Read(rBuff)
		if err != nil {
			err = fmt.Errorf("Ошибка получения ответа: %v", err)
			return
		}
		if rBuff[0] == eSign {
			break
		}
		buff = append(buff, rBuff[0])
	}
	if num >= numBody {
		err = fmt.Errorf("Ошибка получения ответа: превышено количество символов до конца ответа")
		return
	}

	// парсинг тела ответа

	// показния веса
	weigth, err = strconv.ParseFloat(strings.TrimLeft(string(buff[2:8]), " "), 64)
	if err != nil {
		err = fmt.Errorf("Ошибка конвертации в число показаний индикатора: %v", err)
		return
	}

	// признак стабильности показаний
	unstable = buff[0] != stabSign

	return
}
