/* global $ */

window.comm = function(websocketurl) {

    let ws = null;

    function retry() {
        console.log("RETRY !");

        try {
            ws = new WebSocket(websocketurl);
        } catch(e) {
            //window.setTimeout(retry, 1000);
        }

        ws.onerror = evt => {
            ws.close();
            window.setTimeout(retry, 1000);
        }
        ws.onopen = evt => {
            console.log("WS OPEN");
            ws.onmessage = evt => $("html").trigger("bytearena:stateupdate", JSON.parse(evt.data));
            ws.onclose = function(evt) {
                console.log("WS CLOSE");
                if(ws !== null) {
                    ws.close();
                    ws = null;
                }
                window.setTimeout(retry, 1000);
            }
        }
    }

    retry();
};
