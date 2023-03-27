package dsp

import "context"

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
