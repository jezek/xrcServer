
function log(t) {
	$("#log").prepend("<p>"+t+"</p>");
}

$(function() {
	log("init");
	var tabs = new Tabs("#header .tab");
	$(".touchpad").each(function(){
		$(this).touchpad = new TouchPad(this);
	});
	$(".touchbutton").each(function(){
		$(this).touchbutton= new TouchButton(this);
	});
	if (window.WebSocket) {
		var socket = new window.WebSocket("ws://"+window.location.host+"/ws");
		socket.onopen = function(evt) {
			log("WebSocket connected");
			pad = $("#touchpad");
			pad.on("touchtap", function() {
				//log("pad on: touchtap");
				socket.send(JSON.stringify({
					type: "click",
					data:{
						button: "left"
					}
				}));
			});
			pad.on("touchdoubletap", function() {
				//log("pad on: touchdoubletap");
				socket.send(JSON.stringify({
					type: "click",
					data:{
						button: "right"
					}
				}));
			});
			pad.on("touchmoverelative", function(e, info) {
				//log("pad on: touchmoverelative");
				socket.send(JSON.stringify({
					type: "moverelative",
					data: {
						x: info.dx,
						y: info.dy
					}
				}));
			});
			pad.on("touchscroll", function(e, info) {
				//log("pad on: touchscroll: "+info.dir);
				socket.send(JSON.stringify({
					type: "scroll",
					data: {
						dir: info.dir
					}
				}));
			});
			left = $("#button_left");
			left.on("touchdown", function(){
				log("left on: touchdown");
				$(this).addClass("down");
				socket.send(JSON.stringify({
					type: "down",
					data: {
						button: "left"
					}
				}));
			});
			left.on("touchup", function(){
				log("left on: touchup");
				$(this).removeClass("down");
				socket.send(JSON.stringify({
					type: "up",
					data: {
						button: "left"
					}
				}));
			});
			left.on("touchdownlock", function(){
				//log("left on: touchdownlock");
				$(this).addClass("locked");
			});
			left.on("touchdownunlock", function(){
				//log("left on: touchdownunlock");
				$(this).removeClass("locked");
			});
		};
		socket.onclose = function(evt) {
			log("WebSocket closed");

			pad = $("#touchpad");
			pad.off("touchtap");
			pad.off("touchdoubletap");
			pad.off("touchmoverelative");
			pad.off("touchscroll");

			left = $("#button_left");
			left.off("touchdown");
			left.off("touchup");
			left.off("touchdownlock");
			left.off("touchdownunlock");
		};
		socket.onmessage = function(evt) {
			jsonData = jQuery.parseJSON(evt.data);
			log("WebSocket read: <code>"+jsonData+"</code>");
		};
	}
	else {
		log("Your browser does not support WebSockets.");
	}
});
