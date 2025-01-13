package device

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/byuoitav/common/status"
	"github.com/byuoitav/connpool"
	"github.com/gin-gonic/gin"
)

//----------------------------------------------------------------------------------Video Switching---------------------------------
// {"jsonrpc": "2.0","id": 1234,"method": "Component.Set","params": {"Name": "VideoZone1","Controls": [{"Name": "selector.6","Value": true}]}}$00
// response: {"jsonrpc":"2.0","result":true,"id":1234}{00}

// uses a selector named component in the q-sys designer file
func (dm *DeviceManager) HandlerSetInput(ctx *gin.Context) {
	slog.Debug("HandlerSetInput Start")

	address := ctx.Param("address")
	component := ctx.Param("component")
	origInput := ctx.Param("input")
	dsp := dm.CreateDSP(address)

	// subtract one from the input since the selector component is 0 based and we want everything to start with 1
	newInput, err := strconv.Atoi(origInput)
	if err != nil {
		slog.Error("error converting input parameter to integer", "address", address, "component", component, "input", origInput)

	}

	if newInput < 1 {
		newInput = 0
		slog.Warn("error with input value, should be above 1", "address", address, "component", component, "input", origInput)
	}
	newInput = newInput - 1

	input := strconv.Itoa(newInput)

	slog.Info("parameters received:", "address", address, "component", component, "input", input)

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err = dsp.setInput(c, component, input)
	if err != nil {
		slog.Error("unable to set input", "address", address, "component", component, "input", input, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("input set", "address", address, "component", component, "input", input)

	ctx.JSON(http.StatusOK, status.Input{
		Input: origInput + ":" + component,
	})

	slog.Debug("HandlerSetInput End")
}

func (d *DSP) setInput(ctx context.Context, component string, input string) error {

	req := d.GetComponentSetStatusRequest(ctx)

	req.Params.Name = component

	var controls QSCComponentControlsSet
	controls.Name = fmt.Sprintf("selector.%s", input)
	controls.Value = true

	req.Params.Controls = append(req.Params.Controls, controls)

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}

	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		slog.Info("sending QRC command for", "component", component, "input", input, "error", err)

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to set input: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to set input: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		if strings.Contains(string(resp), "UnknownControls") || strings.Contains(string(resp), "error") {
			return fmt.Errorf("error from Q-Sys DSP: %s", resp)
		}

		slog.Debug("Got response: ", "response", string(resp))

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (dm *DeviceManager) HandlerGetInput(ctx *gin.Context) {
	var err error
	slog.Debug("HandlerGetInput Start")

	address := ctx.Param("address")
	component := ctx.Param("component")
	dsp := dm.CreateDSP(address)

	if address == "" || component == "" {
		slog.Error("invalid address or component", "address", address, "component", component)
		return
	}

	slog.Debug("parameters received:", "address", address, "component", component)

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	source, err := dsp.getInput(c, component)
	if err != nil {
		slog.Error("unable to get input", "address", address, "component", component, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	sourceString := strconv.Itoa(source)
	slog.Debug("input set", "address", address, "component", component, "input", sourceString)

	ctx.JSON(http.StatusOK, status.Input{
		Input: sourceString + ":" + component,
	})

	slog.Debug("HandlerGetInput End")
}

// queries DSP and returns int representing the input for the selector that is a named component
func (d *DSP) getInput(ctx context.Context, component string) (input int, err error) {

	req := d.GetComponentGetStatusRequest(ctx)

	req.Params.Name = component

	// var controls QSCComponentGetStatusParams
	// controls.Name = component

	//req.Params.Controls = append(req.Params.Controls, controls)

	// {"jsonrpc": "2.0","id": 1234,"method": "Component.GetControls","params": {"Name": "VideoZone1"}}$00
	toSend, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	toSend = append(toSend, 0x00)

	var resp []byte
	var respParsed QSCComponentGetStatusResponse
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		slog.Info("sending QRC command for", "component", component, "error", err)

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to getInput: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to getInput: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response getInput: %w", err)
		}

		slog.Debug("Got response: ", "response", string(resp))

		//remove the null termination from the QSC response before unmarshalling
		resp := strings.TrimRight(string(resp), "\x00")
		err = json.Unmarshal([]byte(resp), &respParsed)
		if err != nil {
			return fmt.Errorf("unable to unmarshall\nresponse: %s, \nerror: %w", resp, err)
		}

		input, err = parseInputResponse(respParsed)
		if err != nil {
			return fmt.Errorf("unable to parseInputResponse\nresponse: %s, \nerror: %w", resp, err)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	return input, nil
}

func parseInputResponse(data QSCComponentGetStatusResponse) (input int, err error) {
	//setup default return data
	err = fmt.Errorf("unable to determine input from response")
	input = 0

	for c := range data.Result.Controls {
		if strings.Contains(data.Result.Controls[c].Name, "selector.") && data.Result.Controls[c].String == "true" {
			index := strings.Split(data.Result.Controls[c].Name, ".")

			input, err = strconv.Atoi(index[1])
			if err != nil {
				return 0, fmt.Errorf("unable to parse input number: %w", err)
			}
			return input + 1, nil
		}
	}
	return input, err
}

// ----------------------------------------------------------------------------------Volume---------------------------------
// Uses a gain object named a namedComponent_Gain in the q-sys designer file
// "namedComponent" is the name of the selector object in the q-sys design.  i.e. the suffix "_Gain" is added to the component
// name for the gain object so the pi system kjnows which object to automaticallt reference if not using the named controls
// used with a DSP room type.  This will hopefully make using q-sys video easier in simple rooms by not requiring setting up
// named controls in addition to the named component.
// Volume sent from ther AV-API as 0-100 which needs to scale to a "normal" gain range for a Q-Sys Gain (-40 to 0)
func (dm *DeviceManager) HandlerSetVideoVolume(ctx *gin.Context) {
	slog.Debug("HandlerSetVideoVolume Start")

	address := ctx.Param("address")
	component := ctx.Param("component")
	level := ctx.Param("level")
	dsp := dm.CreateDSP(address)

	// scale level for QSC
	newLevel, err := strconv.Atoi(level)
	if err != nil {
		slog.Error("error converting input parameter to integer", "address", address, "component", component, "input", level)
		ctx.String(http.StatusInternalServerError, err.Error())
	}

	if newLevel < 0 {
		newLevel = 0
		slog.Warn("error with input value, should be above 0", "address", address, "component", component, "input", level)
	} else if newLevel > 100 {
		newLevel = 100
		slog.Warn("error with input value, should be at or below 100", "address", address, "component", component, "input", level)
	}
	scaledLevel := scaleVolume(newLevel)
	slog.Info("parameters received:", "address", address, "component", component, "level", level, "scaledLevel", scaledLevel)

	//QSC Communication
	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err = dsp.setVolume(c, component, scaledLevel)
	if err != nil {
		slog.Error("unable to set input", "address", address, "component", component, "level", level, "scaledLevel", scaledLevel, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("input set", "address", address, "component", component, "level", level, "scaledLevel", scaledLevel)

	ctx.JSON(http.StatusOK, status.Input{
		Input: strconv.Itoa(newLevel),
	})

	slog.Debug("HandlerSetVideoVolume End")
}

func (d *DSP) setVolume(ctx context.Context, component string, level string) error {

	req := d.GetComponentSetStatusRequest(ctx)

	if strings.Contains(component, "_Gain") {
		req.Params.Name = component
	} else {
		req.Params.Name = component + "_Gain"
	}

	var controls QSCComponentControlsSet
	controls.Name = "gain"
	controls.Value = level

	req.Params.Controls = append(req.Params.Controls, controls)

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}

	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		slog.Info("sending QRC command for", "component", req.Params.Name, "level", level, "error", err)

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to set input: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to set input: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response: %w", err)
		}

		if strings.Contains(string(resp), "UnknownControls") || strings.Contains(string(resp), "error") {
			return fmt.Errorf("error from Q-Sys DSP: %s", resp)
		}
		slog.Debug("Got response: ", "response", string(resp))

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (dm *DeviceManager) HandlerGetVideoVolume(ctx *gin.Context) {
	slog.Debug("HandlerGetVideoVolume Start")

	address := ctx.Param("address")
	component := ctx.Param("component")
	dsp := dm.CreateDSP(address)

	if address == "" || component == "" {
		slog.Error("invalid address or component", "address", address, "component", component)
		return
	}

	slog.Debug("parameters received:", "address", address, "component", component)

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	volume, err := dsp.getVolume(c, component)
	if err != nil {
		slog.Error("unable to get input", "address", address, "component", component, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	slog.Debug("input set", "address", address, "component", component, "volume", volume)

	ctx.JSON(http.StatusOK, status.Volume{
		Volume: volume,
	})

	slog.Debug("HandlerGetVideoVolume End")
}

// queries DSP and returns int representing the input for the selector that is a named component
// format of query: {"jsonrpc": "2.0","id": 1234,"method": "Component.GetControls","params": {"Name": "VideoZone1"}}$00
func (d *DSP) getVolume(ctx context.Context, component string) (volume int, err error) {

	req := d.GetComponentGetStatusRequest(ctx)

	if strings.Contains(component, "_Gain") {
		req.Params.Name = component
	} else {
		req.Params.Name = component + "_Gain"
	}

	toSend, err := json.Marshal(req)
	if err != nil {
		return
	}

	toSend = append(toSend, 0x00)

	var resp []byte
	var respParsed QSCComponentGetStatusResponse
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		slog.Info("sending QRC command for", "component", component, "error", err)

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to getVolume: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to getVolume: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response getVolume: %w", err)
		}

		slog.Debug("Got response: ", "response", string(resp))

		//remove the null termination from the QSC response before unmarshalling
		resp := strings.TrimRight(string(resp), "\x00")
		err = json.Unmarshal([]byte(resp), &respParsed)
		if err != nil {
			return fmt.Errorf("unable to unmarshall\nresponse: %s, \nerror: %w", resp, err)
		}

		gainIndex := 0
		found := false
		for i := range respParsed.Result.Controls {
			if respParsed.Result.Controls[i].Name == "gain" {
				gainIndex = i
				found = true
			}
		}
		if found == false {
			return fmt.Errorf("gain not found in response\nresponse: %s", resp)
		}

		slog.Debug("Got value: ", "value", respParsed.Result.Controls[gainIndex].Value)

		volConv, ok := respParsed.Result.Controls[gainIndex].Value.(float64)
		if ok {
			slog.Debug("successfully converted gain value to int")
			volume = int(volConv)
		} else {
			slog.Error("error converting gain value to int")
		}

		return nil
	})
	if err != nil {
		return
	}

	volume = scaleReceiveVolume(volume)
	return volume, nil
}

func scaleVolume(volume int) string {
	v := float64(volume) //make a float64 for accuracy otherwise 50 returns 49
	if v > 100 {
		v = 100
	}
	if v < 1 {
		v = 0
	}
	outMax := 100.0
	outMin := 0.0
	devHi := 0.0
	devLo := -40.0
	mutedLevel := -100.0

	vol := ((devHi-devLo)*(v-outMin))/(outMax-outMin) + devLo
	if v < 1 {
		vol = mutedLevel
	}

	volToSend := int(vol)
	return fmt.Sprint(volToSend)
}

func scaleReceiveVolume(vtmp int) (volToSend int) {
	// //vtmp, err := strconv.Atoi(volume)
	// if err != nil {
	// 	return "0"
	// }
	v := float64(vtmp) //make a float64 for accuracy otherwise 50 returns 49

	outMax := 100.0
	outMin := 0.0
	devHi := 0.0
	devLo := -40.0

	if v < devLo {
		volToSend = int(outMin)
		return
	}

	vol := ((outMax-outMin)*(v-devLo))/(devHi-devLo) + outMin
	volToSend = int(vol)
	if volToSend > 100 {
		volToSend = 100
	}
	if volToSend < 1 {
		volToSend = 0
	}
	return
}

// ----------------------------------------------------------------------------------Mute---------------------------------
// Handler to set the mute status for the video component
func (dm *DeviceManager) HandlerSetVideoMute(ctx *gin.Context) {
	slog.Debug("HandlerSetVideoMute Start")

	address := ctx.Param("address")     // Device/DSP address
	component := ctx.Param("component") // Device/component name
	mute := ctx.Param("mute")           // Mute boolean parameter (should be "true" or "false")

	// Parse the mute parameter into a boolean
	muteStatus, err := strconv.ParseBool(mute)
	if err != nil {
		slog.Error("error converting mute parameter to boolean", "address", address, "component", component, "mute", mute)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Communicate with the DSP to set the mute status
	dsp := dm.CreateDSP(address)

	slog.Info("parameters received:", "address", address, "component", component, "mute", muteStatus)

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err = dsp.setMute(c, component, muteStatus)
	if err != nil {
		slog.Error("unable to set mute", "address", address, "component", component, "mute", muteStatus, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	slog.Debug("mute set", "address", address, "component", component, "mute", muteStatus)

	ctx.JSON(http.StatusOK, status.Mute{
		Muted: muteStatus,
	})

	slog.Debug("HandlerSetVideoMute End")
}

// DSP method to set the mute status for a component
func (d *DSP) setMute(ctx context.Context, component string, mute bool) error {
	req := d.GetComponentSetStatusRequest(ctx)

	if strings.Contains(component, "_Gain") {
		req.Params.Name = component
	} else {
		req.Params.Name = component + "_Gain"
	}

	var controls QSCComponentControlsSet
	controls.Name = "mute" // Set the name to "mute"
	controls.Value = mute  // Set the mute value (true or false)

	req.Params.Controls = append(req.Params.Controls, controls)

	toSend, err := json.Marshal(req)
	if err != nil {
		return err
	}

	toSend = append(toSend, 0x00)

	var resp []byte
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		slog.Info("sending QRC command for", "component", req.Params.Name, "mute", mute, "error", err)

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

		if strings.Contains(string(resp), "UnknownControls") || strings.Contains(string(resp), "error") {
			return fmt.Errorf("error from Q-Sys DSP: %s", resp)
		}
		slog.Debug("Got response: ", "response", string(resp))

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// Handler to get the current mute status for the video component
func (dm *DeviceManager) HandlerGetVideoMute(ctx *gin.Context) {
	slog.Debug("HandlerGetVideoMute Start")

	address := ctx.Param("address")
	component := ctx.Param("component")

	// Query the DSP for the current mute status
	dsp := dm.CreateDSP(address)

	slog.Debug("parameters received:", "address", address, "component", component)

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	muteStatus, err := dsp.getMute(c, component)
	if err != nil {
		slog.Error("unable to get mute", "address", address, "component", component, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	slog.Debug("current mute status", "address", address, "component", component, "mute", muteStatus)

	ctx.JSON(http.StatusOK, status.Mute{
		Muted: muteStatus,
	})

	slog.Debug("HandlerGetVideoMute End")
}

// DSP method to get the current mute status for a component
func (d *DSP) getMute(ctx context.Context, component string) (mute bool, err error) {
	req := d.GetComponentGetStatusRequest(ctx)

	if strings.Contains(component, "_Gain") {
		req.Params.Name = component
	} else {
		req.Params.Name = component + "_Gain"
	}

	toSend, err := json.Marshal(req)
	if err != nil {
		return false, err
	}

	toSend = append(toSend, 0x00)

	var resp []byte
	var respParsed QSCComponentGetStatusResponse
	err = d.pool.Do(ctx, func(conn connpool.Conn) error {
		slog.Info("sending QRC command for", "component", component, "error", err)

		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		n, err := conn.Write(toSend)
		switch {
		case err != nil:
			return fmt.Errorf("unable to write command to getMute: %v", err)
		case n != len(toSend):
			return fmt.Errorf("unable to write command to getMute: wrote %v/%v bytes", n, len(toSend))
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return fmt.Errorf("no deadline set")
		}

		resp, err = conn.ReadUntil('\x00', deadline)
		if err != nil {
			return fmt.Errorf("unable to read response getMute: %w", err)
		}

		slog.Debug("Got response: ", "response", string(resp))

		// Remove the null termination from the QSC response before unmarshalling
		resp := strings.TrimRight(string(resp), "\x00")
		err = json.Unmarshal([]byte(resp), &respParsed)
		if err != nil {
			return fmt.Errorf("unable to unmarshall\nresponse: %s, \nerror: %w", resp, err)
		}

		// Extract mute status from the response
		for _, control := range respParsed.Result.Controls {
			if control.Name == "mute" {
				mute, _ = control.Value.(bool)
				return nil
			}
		}

		return fmt.Errorf("mute not found in response\nresponse: %s", resp)
	})
	if err != nil {
		return false, err
	}

	return mute, nil
}
