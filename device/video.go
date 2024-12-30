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

	slog.Debug("parameters received:", "address", address, "component", component, "input", input)

	c, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	err = dsp.SetInput(c, component, input)
	if err != nil {
		slog.Error("unable to set input", "address", address, "component", component, "input", input, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	slog.Debug("input set", "address", address, "component", component, "input", input)

	ctx.JSON(http.StatusOK, status.Input{
		Input: origInput,
	})

	slog.Debug("HandlerSetInput End")
}

func (d *DSP) SetInput(ctx context.Context, component string, input string) error {

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

		slog.Debug("Got response: ", "response", string(resp))

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// ********************************************************************Working Here
// ********************************************************************Working Here
// ********************************************************************Working Here

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

	source, err := dsp.GetInput(c, component)
	if err != nil {
		slog.Error("unable to get input", "address", address, "component", component, "error", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	sourceString := strconv.Itoa(source)
	slog.Debug("input set", "address", address, "component", component, "input", sourceString)

	ctx.JSON(http.StatusOK, status.Input{
		Input: sourceString,
	})

	slog.Debug("HandlerGetInput End")
}

// queries DSP and returns int representing the input for the selector that is a named component
func (d *DSP) GetInput(ctx context.Context, component string) (input int, err error) {

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

		slog.Debug("Got response: ", "response", string(resp))

		//remove the null termination from the QSC response before unmarshalling
		resp := strings.TrimRight(string(resp), "\x00")
		err = json.Unmarshal([]byte(resp), &respParsed)
		if err != nil {
			return fmt.Errorf("unable to unmarshall\nresponse: %s, \nerror: %w", resp, err)
		}

		input, err = ParseInputResponse(respParsed)
		if err != nil {
			return fmt.Errorf("unable to ParseInputResponse\nresponse: %s, \nerror: %w", resp, err)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	return input, nil
}

func ParseInputResponse(data QSCComponentGetStatusResponse) (input int, err error) {
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
func (dm *DeviceManager) HandlerSetVideoVolume(ctx *gin.Context) {
	slog.Debug("HandlerSetVideoVolume Start")

	address := ctx.Param("address")
	component := ctx.Param("component") + "_Gain"
	level := ctx.Param("level")

	slog.Debug("parameters received:", "address", address, "component", component, "level", level)

	slog.Debug("HandlerSetVideoVolume End")
}

func (dm *DeviceManager) HandlerGetVideoVolume(ctx *gin.Context) {
	slog.Debug("HandlerGetVideoVolume Start")

	address := ctx.Param("address")
	component := ctx.Param("component") + "_Gain"

	slog.Debug("parameters received:", "address", address, "component", component)

	slog.Debug("HandlerGetVideoVolume End")
}

// ----------------------------------------------------------------------------------Mute---------------------------------
func (dm *DeviceManager) HandlerSetVideoMute(ctx *gin.Context) {
	slog.Debug("HandlerSetVideoMute Start")

	address := ctx.Param("address") //device/DSP address
	component := ctx.Param("component") + "_Gain"
	mute := ctx.Param("mute") //boolean

	slog.Debug("parameters received:", "address", address, "component", component, "mute", mute)

	slog.Debug("HandlerSetVideoMute End")
}

func (dm *DeviceManager) HandlerGetVideoMute(ctx *gin.Context) {
	slog.Debug("HandlerGetVideoMute Start")

	address := ctx.Param("address")
	component := ctx.Param("component") + "_Gain"

	slog.Debug("parameters received:", "address", address, "component", component)

	slog.Debug("HandlerGetVideoMute End")
}

//get component info command
// {"jsonrpc": "2.0","id": 1234,"method": "Component.GetControls","params": {"Name": "VideoZone1"}}$00
//response:

// {
//     "jsonrpc": "2.0",
//     "result": {
//         "Name": "VideoZone1",
//         "Controls": [
//             {
//                 "Name": "label.0",
//                 "Type": "Text",
//                 "String": "Selection 1",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.1",
//                 "Type": "Text",
//                 "String": "Selection 2",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.2",
//                 "Type": "Text",
//                 "String": "Selection 3",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.3",
//                 "Type": "Text",
//                 "String": "Selection 4",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.4",
//                 "Type": "Text",
//                 "String": "Selection 5",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.5",
//                 "Type": "Text",
//                 "String": "Selection 6",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.6",
//                 "Type": "Text",
//                 "String": "Selection 7",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.7",
//                 "Type": "Text",
//                 "String": "Selection 8",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.8",
//                 "Type": "Text",
//                 "String": "Selection 9",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "label.9",
//                 "Type": "Text",
//                 "String": "Selection 10",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector",
//                 "Type": "Text",
//                 "String": "{\"Text\":\"Selection 5\",\"TextColor\":\"\",\"Icon\":\"\",\"IconColor\":\"\",\"Data\":\"Value 5\",\"Index\":4}",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.0",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.1",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.2",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.3",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.4",
//                 "Type": "Boolean",
//                 "Value": true,
//                 "String": "true",
//                 "Position": 1.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.5",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.6",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.7",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.8",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "selector.9",
//                 "Type": "Boolean",
//                 "Value": false,
//                 "String": "false",
//                 "Position": 0.0,
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value",
//                 "Type": "Text",
//                 "String": "Value 5",
//                 "Direction": "Read Only"
//             },
//             {
//                 "Name": "value.0",
//                 "Type": "Text",
//                 "String": "Value 1",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.1",
//                 "Type": "Text",
//                 "String": "Value 2",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.2",
//                 "Type": "Text",
//                 "String": "Value 3",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.3",
//                 "Type": "Text",
//                 "String": "Value 4",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.4",
//                 "Type": "Text",
//                 "String": "Value 5",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.5",
//                 "Type": "Text",
//                 "String": "Value 6",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.6",
//                 "Type": "Text",
//                 "String": "Value 7",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.7",
//                 "Type": "Text",
//                 "String": "Value 8",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.8",
//                 "Type": "Text",
//                 "String": "Value 9",
//                 "Direction": "Read/Write"
//             },
//             {
//                 "Name": "value.9",
//                 "Type": "Text",
//                 "String": "Value 10",
//                 "Direction": "Read/Write"
//             }
//         ]
//     },
//     "id": 1234
// }{
//     00
// }
