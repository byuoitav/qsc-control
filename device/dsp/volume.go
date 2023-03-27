package dsp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/byuoitav/connpool"
	"go.uber.org/zap"
)

func (d *DSP) Volumes(ctx context.Context, blocks []string) (map[string]int, error) {
	toReturn := make(map[string]int)

	for i, block := range blocks {
		req := d.GetGenericGetStatusRequest(ctx)
		req.Params = append(req.Params, block)

		qscResp := QSCGetStatusResponse{}

		toSend, err := json.Marshal(req)
		if err != nil {
			return toReturn, err
		}
		toSend = append(toSend, 0x00)

		var resp []byte

		err = d.pool.Do(ctx, func(conn connpool.Conn) error {
			d.log.Info("Getting volume on %v", zap.String("block", block))
			conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

			n, err := conn.Write(toSend)
			switch {
			case err != nil:
				return fmt.Errorf("unable to write command to get volume for block %v: %v", block, err)
			case n != len(toSend):
				return fmt.Errorf("unable to write command to get volume for block %v: wrote %v/%v bytes", block, n, len(toSend))
			}

			deadline, ok := ctx.Deadline()
			if !ok {
				return fmt.Errorf("no deadline set")
			}

			resp, err = conn.ReadUntil(byte('\x00'), deadline)
			if err != nil {
				return fmt.Errorf("unable to read response: %w", err)
			}

			d.log.Debug("Got response: %v", zap.Any("response", resp))
			fmt.Printf("resp: %s\n", resp)

			return nil
		})
		if err != nil {
			return toReturn, err
		}
		resp = bytes.Trim(resp, "\x00")
		err = json.Unmarshal(resp, &qscResp)
		if err != nil {
			log.Printf(err.Error())
			return toReturn, fmt.Errorf("error unmarshaling response: %v", err)
		}

		log.Printf("[QSC-Communication] Response received: %+v\n", qscResp)

		//get the volume out of the dsp and run it through our equation to reverse it
		found := false
		for _, res := range qscResp.Result {
			if res.Name == block {
				toReturn[blocks[i]] = d.DbToVolumeLevel(ctx, res.Value)
				found = true
				break
			}
		}
		if found {
			continue
		}

		return toReturn, errors.New("[QSC-Communication] No value returned with the name matching the requested state")
	}

	return toReturn, nil
}

func (d *DSP) Mutes(ctx context.Context, blocks []string) (map[string]bool, error) {
	toReturn := make(map[string]bool)

	for i, block := range blocks {
		req := d.GetGenericGetStatusRequest(ctx)
		req.Params = append(req.Params, block)

		qscResp := QSCGetStatusResponse{}

		toSend, err := json.Marshal(req)
		if err != nil {
			return toReturn, err
		}
		toSend = append(toSend, 0x00)

		var resp []byte

		err = d.pool.Do(ctx, func(conn connpool.Conn) error {
			d.log.Info("Getting mute on %v", zap.String("block", block))
			conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

			n, err := conn.Write(toSend)
			switch {
			case err != nil:
				return fmt.Errorf("unable to write command to get mute for block %v: %v", block, err)
			case n != len(toSend):
				return fmt.Errorf("unable to write command to get mute for block %v: wrote %v/%v bytes", block, n, len(toSend))
			}

			deadline, ok := ctx.Deadline()
			if !ok {
				return fmt.Errorf("no deadline set")
			}

			resp, err = conn.ReadUntil('\x00', deadline)
			if err != nil {
				return fmt.Errorf("unable to read response: %w", err)
			}

			d.log.Debug("Got response: %v", zap.Any("response", resp))

			return nil
		})
		if err != nil {
			return toReturn, err
		}

		resp = bytes.Trim(resp, "\x00")
		err = json.Unmarshal(resp, &qscResp)
		if err != nil {
			log.Printf(err.Error())
			return toReturn, err
		}

		//get the volume out of the dsp and run it through our equation to reverse it
		found := false
		for _, res := range qscResp.Result {
			if res.Name == block {
				if res.Value == 1.0 {
					toReturn[blocks[i]] = true
					found = true
				}
				if res.Value == 0.0 {
					toReturn[blocks[i]] = false
					found = true
				}
				break
			}
		}
		if found {
			continue
		}

		errmsg := "[QSC-Communication] No value returned with the name matching the requested state"
		log.Printf(errmsg)
		return toReturn, errors.New(errmsg)
	}

	return toReturn, nil
}

func (d *DSP) SetVolume(ctx context.Context, block string, volume int) error {

	log.Printf("got: %v", volume)
	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = block

	if volume == 0 {
		req.Params.Value = -100
	} else {
		//do the logarithmic magic
		req.Params.Value = d.VolToDb(ctx, volume)
	}
	log.Printf("sending: %v", req.Params.Value)

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}
	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		d.log.Info("setting volume on %v to %v", zap.String("block", block), zap.Int("level", volume))

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to set volume: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to set volume: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		d.log.Debug("Got response: %v", zap.Any("response", resp))

		return nil
	})
	if err != nil {
		return err
	}

	//we need to unmarshal our response, parse it for the value we care about, then role with it from there
	qscResp := QSCSetStatusResponse{}
	resp = bytes.Trim(resp, "\x00")
	err = json.Unmarshal(resp, &qscResp)
	if err != nil {
		log.Printf("Error: %v", err.Error())
		return err
	}
	if qscResp.Result.Name != block {
		errmsg := fmt.Sprintf("Invalid response, the name recieved does not match the name sent %v/%v", block, qscResp.Result.Name)
		log.Printf(errmsg)
		return errors.New(errmsg)
	}

	return nil
}

func (d *DSP) SetMute(ctx context.Context, block string, mute bool) error {

	//we generate our set status request, then we ship it out

	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = block
	if mute {
		req.Params.Value = 1
	} else {
		req.Params.Value = 0
	}

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}
	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		d.log.Info("setting mute on %v to %v", zap.String("block", block), zap.Bool("mute", mute))

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to set mute: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to set mute: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		d.log.Debug("Got response: %v", zap.Any("response", resp))

		return nil
	})
	if err != nil {
		return err
	}

	//we need to unmarshal our response, parse it for the value we care about, then role with it from there
	qscResp := QSCSetStatusResponse{}
	resp = bytes.Trim(resp, "\x00")
	err = json.Unmarshal(resp, &qscResp)
	if err != nil {
		log.Printf("Error: %v", err.Error())
		return err
	}

	//otherwise we check to see what the value is set to
	if qscResp.Result.Name != block {
		errmsg := fmt.Sprintf("Invalid response, the name recieved does not match the name sent %v/%v", block, qscResp.Result.Name)
		log.Printf(errmsg)
		return errors.New(errmsg)
	}

	if qscResp.Result.Value == 1.0 {
		return nil
	}
	if qscResp.Result.Value == 0.0 {
		return nil
	}
	errmsg := fmt.Sprintf("[QSC-Communication] Invalid response received: %v", qscResp.Result)
	log.Printf(errmsg)
	return errors.New(errmsg)
}

func (d *DSP) DbToVolumeLevel(ctx context.Context, level float64) int {
	return int(math.Pow(10, (level/20)) * 100)
}

func (d *DSP) VolToDb(ctx context.Context, level int) float64 {
	return math.Log10(float64(level)/100) * 20
}
