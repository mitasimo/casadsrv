package main

import (
	"context"
	"encoding/json"
	"net/http"
	"simd/winservice"
	"time"
)

// DeviceHandler содержит методы, которые
type DeviceHandler interface {
	GetWeigth() (float64, bool, error)
	Disconnect()
}

// NewService -
func NewService(host string, device DeviceHandler) *Service {
	return &Service{host: host, device: device}
}

// Service -
type Service struct {
	host       string
	server     *http.Server
	stopResult chan error
	device     DeviceHandler
}

// Start -
func (s *Service) Start() error {
	var err error

	// create router
	srvMux := http.NewServeMux()

	// add device as root handler
	srvMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		weigth, unstable, err := s.device.GetWeigth()
		var res []byte
		if err != nil {
			res = errorResult(err)
		} else {
			res = successResult(weigth, unstable)
		}
		w.Write(res)
	})

	// create http server
	server := &http.Server{
		Addr:         s.host,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		Handler:      srvMux,
	}

	// канал для получения результата выполнения server.ListenAndServe()
	stopResult := make(chan error, 1)

	// запустить http сервер
	go func() {
		defer close(stopResult)
		stopResult <- server.ListenAndServe()
	}()

	select {
	case err = <-stopResult:
	case <-time.After(time.Second * 2): // ожидать запуск http-сервера 2 секунды
		err = nil
	}

	if err != nil {
		s.device.Disconnect()
		if err == http.ErrServerClosed {
			err = nil
		}
	} else {
		s.server = server
		s.stopResult = stopResult
	}

	return err
}

// Stop -
func (s *Service) Stop() error {
	defer s.device.Disconnect()

	var err error
	// начать остановку http-сервера
	err = s.server.Shutdown(context.Background())
	if err != nil {
		winservice.LogWarningf("Error server.Shutdown(): %v", err)
	}
	err = <-s.stopResult // получить результат ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		winservice.LogWarningf("Error server.ListenAndServe(): %v", err)
	}

	return err
}

func errorResult(err error) []byte {
	return result(err, 0, false)
}

func successResult(weigth float64, unstable bool) []byte {
	return result(nil, weigth, unstable)
}

func result(err error, weigth float64, unstable bool) []byte {
	type rw struct {
		HasError         bool    `json:"error"`
		ErrorDescription string  `json:"description,omitempty"`
		Weigth           float64 `json:"weigth"`
		Unstable         bool    `json:"unstable"`
	}

	r := &rw{}
	if err != nil {
		r.HasError = true
		r.ErrorDescription = err.Error()
	} else {
		r.Weigth = weigth
	}
	r.Unstable = unstable

	toJSON, _ := json.Marshal(r)
	return toJSON
}
