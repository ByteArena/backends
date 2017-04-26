/* global $ */

window.comm = function(websocketurl) {

    const ws = new WebSocket(websocketurl);

    ws.onopen = evt => console.log("WS OPEN");
    ws.onerror = evt => console.error("WS ERROR", evt);
    ws.onmessage = evt => $("html").trigger("bytearena:stateupdate", JSON.parse(evt.data));
    ws.onclose = function(evt) {
        console.log("WS CLOSE");
        ws.close();
        ws = null;
    }
};
