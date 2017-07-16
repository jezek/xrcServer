
function log(t) {
	$("#content .log").prepend("<p>"+t+"</p>");
};

$(function() {
	var pad = new TouchPad("#touchpad");
	var left = new TouchButton("#button_left");
	if (window["WebSocket"]) {
		var socket = new WebSocket("ws://"+window.location.host+"/ws");
		socket.onopen = function(evt) {
			log("WebSocket connected");
			pad.elm.on("touchtap", function() {
				//log("pad.elm on: touchtap");
				socket.send(JSON.stringify({
					type: "click",
					data:{
						button: "left"
					}
				}));
			});
			pad.elm.on("touchdoubletap", function() {
				//log("pad.elm on: touchdoubletap");
				socket.send(JSON.stringify({
					type: "click",
					data:{
						button: "right"
					}
				}));
			});
			pad.elm.on("touchmoverelative", function(e, info) {
				//log("pad.elm on: touchmoverelative");
				socket.send(JSON.stringify({
					type: "moverelative",
					data: {
						x: info.dx,
						y: info.dy
					}
				}));
			});
			pad.elm.on("touchscroll", function(e, info) {
				//log("pad.elm on: touchscroll: "+info.dir);
				socket.send(JSON.stringify({
					type: "scroll",
					data: {
						dir: info.dir
					}
				}));
			});
		};
		socket.onclose = function(evt) {
			log("WebSocket closed");
			pad.elm.off("touchtap");
			pad.elm.off("touchdoubletap");
			pad.elm.off("touchmoverelative");
			pad.elm.off("touchscroll");
		}
		socket.onmessage = function(evt) {
			jsonData = jQuery.parseJSON(evt.data);
			log("WebSocket read: <code>"+jsonData+"</code>");
		}
	}
	else {
		log("Your browser does not support WebSockets.");
	}
});
