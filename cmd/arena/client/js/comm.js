/* global $ */

window.comm = function(websocketurl) {

    var $output = $("#output");
    var $open = $("#open");
    var $close = $("#close");
    var ws;

    var print = function(message) {
        var d = $("<div></div>");
        d.html(message);
        output.append(d.get(0));
    };

    $open.click(function(e) {

        if (ws) return;

        ws = new WebSocket(websocketurl);

        ws.onopen = function(evt) {
            print("OPEN");
            $open.hide();
            $close.show();
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws.close();
            ws = null;
            $open.show();
            $close.hide();
        }

        ws.onmessage = function(evt) {
            window.onStateUpdate(JSON.parse(evt.data));
        }

        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
    });

    $close.click(function(e) {
        if (!ws) return;
        ws.close();
    });

};
