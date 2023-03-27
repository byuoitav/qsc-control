package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/byuoitav/connpool"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (dm *DeviceManager) HandlerGetGeneric(ctx *gin.Context) {}

func (dm *DeviceManager) HandlerSetGeneric(ctx *gin.Context) {}

func (d *DSP) SetControl(ctx context.Context, name string, value float64) error {
	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = name
	req.Params.Value = value

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}
	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		d.log.Info("Setting control", zap.String("name", name), zap.Float64("value", value))

		deadline, ok := ctx.Deadline()
		if !ok {
			deadline = time.Now().Add(3 * time.Second)
		}

		conn.SetWriteDeadline(deadline)

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to set volume: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to set volume: wrote %v/%v bytes", n, len(toSend))
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		d.log.Debug("Got response", zap.ByteString("response", resp))
		return nil
	})
	if err != nil {
		return err
	}

	resp = bytes.Trim(resp, "\x00")

	qscResp := QSCSetStatusResponse{}
	if err := json.Unmarshal(resp, &qscResp); err != nil {
		return fmt.Errorf("unable to parse response: %w", err)
	}

	if qscResp.Result.Name != name {
		return fmt.Errorf("response name (%s) does not match the name sent (%s)", qscResp.Result.Name, name)
	}

	return nil
}

func (d *DSP) Control(ctx context.Context, name string) (float64, error) {
	req := d.GetGenericGetStatusRequest(ctx)
	req.Params = append(req.Params, name)

	toSend, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		d.log.Info("Getting control", zap.String("name", name))

		deadline, ok := ctx.Deadline()
		if !ok {
			deadline = time.Now().Add(3 * time.Second)
		}

		conn.SetWriteDeadline(deadline)

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command: %w", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command: wrote %v/%v bytes", n, len(toSend))
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		d.log.Debug("Got response", zap.ByteString("response", resp))
		return nil
	})
	if err != nil {
		return 0, err
	}

	resp = bytes.Trim(resp, "\x00")
	qscResp := QSCGetStatusResponse{}
	if err := json.Unmarshal(resp, &qscResp); err != nil {
		return 0, fmt.Errorf("unable to parse response: %w", err)
	}

	if len(qscResp.Result) == 0 {
		return 0, fmt.Errorf("no results in response: '%s'", resp)
	}

	return qscResp.Result[0].Value, nil
}
