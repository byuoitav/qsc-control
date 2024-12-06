package device

import "github.com/gin-gonic/gin"

//set selector command
// {"jsonrpc": "2.0","id": 1234,"method": "Component.Set","params": {"Name": "VideoZone1","Controls": [{"Name": "selector.6","Value": true}]}}$00
//response: {"jsonrpc":"2.0","result":true,"id":1234}{00}
func (dm *DeviceManager) HandlerSetInput(ctx *gin.Context) {

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

func (dm *DeviceManager) HandlerGetInput(ctx *gin.Context) {

}
