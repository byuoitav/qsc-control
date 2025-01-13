package device

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/byuoitav/connpool"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DeviceManager struct {
	Log      *zap.Logger
	LogLevel *zap.AtomicLevel
	DspList  *sync.Map
}

func (dm *DeviceManager) RunHTTPServer(router *gin.Engine, port string) error {
	dm.Log.Info("registering http endpoints")

	dev := router.Group("")
	dev.GET("/:address/:name/volume/mute", dm.HandlerMute)
	dev.GET("/:address/:name/volume/unmute", dm.HandlerUnMute)
	dev.GET("/:address/:name/mute/status", dm.HandlerMuteStatus)
	dev.GET("/:address/:name/volume/set/:level", dm.HandlerSetVolume)
	dev.GET("/:address/:name/volume/level", dm.HandlerGetVolume)
	dev.PUT("/:address/generic/:name/:value", dm.HandlerSetGeneric)
	dev.GET("/:address/generic/:name", dm.HandlerGetGeneric)
	dev.GET("/:address/hardware", dm.HandlerGetInfo)
	dev.GET("/:address/health", dm.HandlerGetHealth)

	video := router.Group("/api/v1/video")
	video.GET("/:address/component/:component/input/:input", dm.HandlerSetInput) //change input
	video.GET("/:address/component/:component/input", dm.HandlerGetInput)        //get input

	video.GET("/:address/component/:component/volume/:level", dm.HandlerSetVideoVolume) //set volume
	video.GET("/:address/component/:component/volume", dm.HandlerGetVideoVolume)        //get volume

	video.GET("/:address/component/:component/mute/:mute", dm.HandlerSetVideoMute) //set mute true/false
	video.GET("/:address/component/:component/muted", dm.HandlerGetVideoMute)      //get mute state
	video.GET("/:address/health", dm.HandlerGetHealth)

	server := &http.Server{
		Addr: port,
		//MaxHeaderBytes: 1024 * 10,
	}

	dm.Log.Info("running http server", zap.String("port", port))
	return router.Run(server.Addr)
}

func (dm *DeviceManager) CreateDSP(addr string) *DSP {
	if dsp, ok := dm.DspList.Load(addr); ok {
		return dsp.(*DSP)
	}

	dsp := newDSP(addr)

	dm.DspList.Store(addr, dsp)
	return dsp
}

type DSP struct {
	pool *connpool.Pool
	log  *zap.Logger
}

//const _kTimeoutInSeconds = 2.0

func newDSP(addr string, opts ...Option) *DSP {
	options := options{
		ttl:    30 * time.Second,
		delay:  500 * time.Millisecond,
		logger: zap.NewNop(),
	}

	for _, o := range opts {
		o.apply(&options)
	}

	d := &DSP{
		pool: &connpool.Pool{
			TTL:    options.ttl,
			Delay:  options.delay,
			Logger: options.logger.Sugar(),
		},
		log: options.logger,
	}

	d.pool.NewConnection = func(ctx context.Context) (net.Conn, error) {
		dial := net.Dialer{}
		conn, err := dial.DialContext(ctx, "tcp", addr+":1710")
		if err != nil {
			return nil, err
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			deadline = time.Now().Add(5 * time.Second)
		}

		conn.SetDeadline(deadline)

		buf := make([]byte, 64)
		for i := range buf {
			buf[i] = 0x01
		}
		for !bytes.Contains(buf, []byte{0x00}) {
			_, err := conn.Read(buf)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("unable to read new connection prompt: %w", err)
			}
		}

		return conn, nil
	}

	return d
}

// BaseRequest are the common parts of every qsc jsonrpc request
type BaseRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
}

type QSCStatusReport struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Platform    string `json:"Platform"`
		State       string `json:"State"`
		DesignName  string `json:"DesignName"`
		DesignCode  string `json:"DesignCode"`
		IsRedundant bool   `json:"IsRedundant"`
		IsEmulator  bool   `json:"IsEmulator"`
		Status      struct {
			Code   int    `json:"Code"`
			String string `json:"String"`
		} `json:"Status"`
	} `json:"params"`
}

type QSCGetStatusResponse struct {
	BaseRequest
	Result []QSCGetStatusResult `json:"result"`
}
type QSCGetStatusResult struct {
	Name     string
	Value    float64
	String   string
	Position float64
}

// QSCStatusGetResponse is the values that we are getting back from the StatusGet method
type QSCStatusGetResult struct {
	Platform    string `json:"Platform"`
	State       string `json:"State"`
	DesignName  string `json:"DesignName"`
	DesignCode  string `json:"DesignCode"`
	IsRedundant bool   `json:"IsRedundant"`
	IsEmulator  bool   `json:"IsEmulator"`
	Status      struct {
		Code   int    `json:"Code"`
		String string `json:"String"`
	} `json:"Status"`
}

type QSCGetStatusRequest struct {
	BaseRequest
	Params []string `json:"params"`
}

type QSCSetStatusRequest struct {
	BaseRequest
	Params QSCSetStatusParams `json:"params"`
}

// QSCSetStatusParams is the parameters for the Control.Set method
type QSCSetStatusParams struct {
	Name  string
	Value float64
}

type QSCSetStatusResponse struct {
	BaseRequest
	Result QSCGetStatusResult `json:"result"`
}

// QSCStatusGetRequest is for the StatusGet method
type QSCStatusGetRequest struct {
	BaseRequest
	Params int `json:"params"`
}

// QSCStatusGetResponse gets the JSON response after calling the StatusGet method
type QSCStatusGetResponse struct {
	BaseRequest
	Result QSCStatusGetResult `json:"result"`
}

func (d *DSP) GetGenericSetStatusRequest(ctx context.Context) QSCSetStatusRequest {
	return QSCSetStatusRequest{BaseRequest: BaseRequest{JSONRPC: "2.0", ID: 1, Method: "Control.Set"}, Params: QSCSetStatusParams{}}
}

func (d *DSP) GetGenericGetStatusRequest(ctx context.Context) QSCGetStatusRequest {
	return QSCGetStatusRequest{BaseRequest: BaseRequest{JSONRPC: "2.0", ID: 1, Method: "Control.Get"}, Params: []string{}}
}

// GetGenericStatusGetRequest is used for retreiving EngineStatus and other information about the QSC
func (d *DSP) GetGenericStatusGetRequest(ctx context.Context) QSCStatusGetRequest {
	return QSCStatusGetRequest{BaseRequest: BaseRequest{JSONRPC: "2.0", ID: 1, Method: "StatusGet"}, Params: 0}
}

//*************************************Video / Named Component Additions***************************************

// Component structs
type QSCComponentSetStatusRequest struct {
	BaseRequest
	Params QSCComponentSetStatusParams `json:"params"`
}

// QSCSetStatusParams is the parameters for the Component.Set method
type QSCComponentSetStatusParams struct {
	Name     string
	Controls []QSCComponentControlsSet
}

// QSCComponentControlsSet is the control paramaters needed to set a control on a selector named component
type QSCComponentControlsSet struct {
	Name  string
	Value interface{}
}

// QSCSetStatusParams is the parameters for the Component.GetControls method
type QSCComponentGetStatusRequest struct {
	BaseRequest
	Params QSCComponentGetStatusParams `json:"params"`
}

// QSCSetStatusParams is the parameters for the Control.Set method
type QSCComponentGetStatusParams struct {
	Name string
}

// QSCComponentGetStatusResponse is the response for the entire component controls from the Component.GetControls method
type QSCComponentGetStatusResponse struct {
	BaseRequest
	Result QSCComponentGetStatusResponseResult `json:"result"`
}

type QSCComponentGetStatusResponseResult struct {
	Name     string
	Controls []QSCComponentGetStatusResponseControls
}

type QSCComponentGetStatusResponseControls struct {
	Name      string      `json:"Name"`
	String    string      `json:"String"`
	Type      string      `json:"Type"`
	Direction string      `json:"Direction"`
	Position  float32     `json:"Position,omitempty"`
	Value     interface{} `json:"Value,omitempty"`
}

func (d *DSP) GetComponentSetStatusRequest(ctx context.Context) QSCComponentSetStatusRequest {
	return QSCComponentSetStatusRequest{BaseRequest: BaseRequest{JSONRPC: "2.0", ID: 1234, Method: "Component.Set"}, Params: QSCComponentSetStatusParams{}}
}

func (d *DSP) GetComponentGetStatusRequest(ctx context.Context) QSCComponentGetStatusRequest {
	return QSCComponentGetStatusRequest{BaseRequest: BaseRequest{JSONRPC: "2.0", ID: 1234, Method: "Component.GetControls"}, Params: QSCComponentGetStatusParams{}}
}
