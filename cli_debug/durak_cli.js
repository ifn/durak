var PORT = 3223;

var COMMANDS = {
    start: 0,
    move: 1
};

//

var ws = new WebSocket("ws://localhost:" + PORT);

var msgStart = {command: COMMANDS.start};
var msgNoCard = {command: COMMANDS.move};
var msgUnknown = {command: 2};

ws.onmessage = function (event) {
    console.log("received:", event.data);
};

ws.sendMsg = function (msg) {
    return function () {
        var jsonMsg = JSON.stringify(msg);
        ws.send(jsonMsg);
        console.log("sent:", jsonMsg);
        return false
    }
}

function sendInputCard () {
    var card = document.getElementById("input_send_card").value;
    var msg = {command: COMMANDS.move, card: card};
    return ws.sendMsg(msg)();
}

//

document.getElementById("btn_go").onclick = ws.sendMsg(msgStart);
document.getElementById("btn_no_card").onclick = ws.sendMsg(msgNoCard);
document.getElementById("frm_send_card").onsubmit = sendInputCard;
