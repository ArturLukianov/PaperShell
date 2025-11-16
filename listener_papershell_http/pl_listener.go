package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	adaptix "github.com/Adaptix-Framework/axc2"
	"github.com/gin-gonic/gin"
)

func (m *ModuleExtender) HandlerListenerValid(data string) error {

	var (
		err  error
		conf HTTPConfig
	)

	// data - это данные из AxScript

	err = json.Unmarshal([]byte(data), &conf)
	if err != nil {
		return err
	}

	if conf.HostBind == "" {
		return errors.New("HostBind is required")
	}

	if conf.PortBind < 1 || conf.PortBind > 65535 {
		return errors.New("PortBind must be in the range 1-65535")
	}

	if conf.CallbackAddress == "" {
		return errors.New("callback_address is required")
	}

	// Check callback address

	host, portStr, err := net.SplitHostPort(conf.CallbackAddress)
	if err != nil {
		return fmt.Errorf("nvalid address (cannot split host:port): %s\n", conf.CallbackAddress)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("Invalid port: %s\n", conf.CallbackAddress)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		if len(host) == 0 || len(host) > 253 {
			return fmt.Errorf("Invalid host: %s\n", conf.CallbackAddress)
		}
		parts := strings.Split(host, ".")
		for _, part := range parts {
			if len(part) == 0 || len(part) > 63 {
				return fmt.Errorf("Invalid host: %s\n", conf.CallbackAddress)
			}
		}
	}

	return nil
}

func (m *ModuleExtender) HandlerCreateListenerDataAndStart(name string, configData string, listenerCustomData []byte) (adaptix.ListenerData, []byte, any, error) {
	var (
		listenerData adaptix.ListenerData // Это то, что будет отображаться в интерфейсе и использоваться агентом
		customdData  []byte
	)

	var (
		listener *HTTP
		conf     HTTPConfig
		err      error
	)

	// listenerCustomData может быть передана вместо конфига - если листенер стартует после перезапуска сервера

	if listenerCustomData == nil {
		// Парсим конфиг - он уже провалидирован, повторно не надо
		err = json.Unmarshal([]byte(configData), &conf)
		if err != nil {
			return listenerData, customdData, listener, err
		}
	} else {
		// Парсим конфиг - он уже провалидирован, повторно не надо
		err = json.Unmarshal(listenerCustomData, &conf)
		if err != nil {
			return listenerData, customdData, listener, err
		}
	}

	// Создаём листенер
	listener = &HTTP{
		GinEngine: gin.New(),
		Name:      name,
		Config:    conf,
		Active:    false,
	}

	// Запускаем листенер
	err = listener.Start(m.ts)
	if err != nil {
		return listenerData, customdData, listener, err
	}

	listenerData = adaptix.ListenerData{
		BindHost:  listener.Config.HostBind,
		BindPort:  strconv.Itoa(listener.Config.PortBind),
		AgentAddr: listener.Config.CallbackAddress,
		Status:    "Listen",
	}

	// Сохраняем конфиг в customdData
	var buffer bytes.Buffer
	err = json.NewEncoder(&buffer).Encode(listener.Config)
	if err != nil {
		return listenerData, customdData, listener, nil
	}
	customdData = buffer.Bytes()

	return listenerData, customdData, listener, nil
}

func (m *ModuleExtender) HandlerEditListenerData(name string, listenerObject any, configData string) (adaptix.ListenerData, []byte, bool) {
	var (
		listenerData adaptix.ListenerData
		customdData  []byte
		ok           bool = false
		err          error
		conf         HTTPConfig
	)

	listener := listenerObject.(*HTTP)
	if listener.Name == name {
		// Parse config
		err = json.Unmarshal([]byte(configData), &conf)
		if err != nil {
			return listenerData, customdData, false
		}

		// Copy from new config to listener
		listener.Config.CallbackAddress = conf.CallbackAddress
		listener.Config.HostBind = conf.HostBind
		listener.Config.PortBind = conf.PortBind

		listenerData = adaptix.ListenerData{
			BindHost:  listener.Config.HostBind,
			BindPort:  strconv.Itoa(listener.Config.PortBind),
			AgentAddr: listener.Config.CallbackAddress,
			Status:    "Listen",
		}

		if !listener.Active {
			listenerData.Status = "Closed"
		}

		var buffer bytes.Buffer
		err = json.NewEncoder(&buffer).Encode(listener.Config)
		if err != nil {
			return listenerData, customdData, false
		}
		customdData = buffer.Bytes()

		ok = true
	}

	return listenerData, customdData, ok
}

func (m *ModuleExtender) HandlerListenerStop(name string, listenerObject any) (bool, error) {
	var (
		err error = nil
		ok  bool  = false
	)

	listener := listenerObject.(*HTTP) // Кастуем к нашему листенеру
	if listener.Name == name {
		err = listener.Stop()
		ok = true
	}

	return ok, err
}

func (m *ModuleExtender) HandlerListenerGetProfile(name string, listenerObject any) ([]byte, bool) {
	var (
		object bytes.Buffer
		ok     bool = false
	)

	listener := listenerObject.(*HTTP)
	if listener.Name == name {
		_ = json.NewEncoder(&object).Encode(listener.Config)
		ok = true
	}

	return object.Bytes(), ok
}
