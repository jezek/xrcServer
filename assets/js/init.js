
function log(t) {
	$("#log").prepend("<p>"+t+"</p>");
}

$(function() {
	log("init");
	var tabs = new Tabs("#header .tab");

	$("#logpage .clear").on("click", function(e) {
		e.preventDefault();
		$("#log").empty();
	});

	if (window.WebSocket) {
		$(".touchpad").each(function(){
			this.touchpad = new TouchPad(this);
		});
		$(".touchbutton").each(function(){
			this.touchbutton= new TouchButton(this);
		});

		var pad = $("#touchpad");
		var left = $("#button_left");
		var right = $("#button_right");

		var socket = new window.WebSocket("ws://"+window.location.host+"/ws");
		socket.onopen = function(evt) {
			log("WebSocket connected");
			pad.on("touchtap", function() {
				log("pad on: touchtap");
				socket.send(JSON.stringify({
					type: "click",
					data:{
						button: "left"
					}
				}));
			});
			pad.on("touchdoubletap", function() {
				log("pad on: touchdoubletap");
				socket.send(JSON.stringify({
					type: "click",
					data:{
						button: "right"
					}
				}));
			});
			pad.on("touchmoverelative", function(e, info) {
				log("pad on: touchmoverelative");
				socket.send(JSON.stringify({
					type: "moverelative",
					data: {
						x: info.dx,
						y: info.dy
					}
				}));
			});
			pad.on("touchscroll", function(e, info) {
				log("pad on: touchscroll: "+info.dir);
				socket.send(JSON.stringify({
					type: "scroll",
					data: {
						dir: info.dir
					}
				}));
			});
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
				log("left on: touchdownlock");
				$(this).addClass("locked");
				$(".touchpad").each(function(){
					this.touchpad.options.tap_enabled = false;
				});
			});
			left.on("touchdownunlock", function(){
				log("left on: touchdownunlock");
				$(this).removeClass("locked");
				$(".touchpad").each(function(){
					this.touchpad.options.tap_enabled = true;
				});
			});

			right.on("touchdown", function(){
				log("right on: touchdown");
				$(this).addClass("down");
				socket.send(JSON.stringify({
					type: "down",
					data: {
						button: "right"
					}
				}));
			});
			right.on("touchup", function(){
				log("right on: touchup");
				$(this).removeClass("down");
				socket.send(JSON.stringify({
					type: "up",
					data: {
						button: "right"
					}
				}));
			});
		};
		socket.onclose = function(evt) {
			log("WebSocket closed");

			pad.off("touchtap");
			pad.off("touchdoubletap");
			pad.off("touchmoverelative");
			pad.off("touchscroll");

			left.off("touchdown");
			left.off("touchup");
			left.off("touchdownlock");
			left.off("touchdownunlock");

			right = $("#button_right");
			right.off("touchdown");
			right.off("touchup");

			tabs.tabs.forEach(function(val, key) {
				$(key).hide();
				val.hide();
			});
			$(tabs.pages.reload.header).show();
			$(tabs.pages.logpage.header).show();
			tabs.select(tabs.pages.reload.header);

			var timer = $("#reload .timer");
			var startTime = parseInt(timer.text()) || 10;
			var interval = null;

			$("#reload .reload").on("click", function(e){
				e.preventDefault();
				if (interval != null) {
					clearInterval(interval);
					interval = null;
				}
				$.ajax(window.location.href, {
					success: function(){
						window.location.reload();
					},
					error:function(){
						log("Server offline");
						var time = startTime;
						timer.html(""+time);
						interval = setInterval(function() {
							time -= 1;
							timer.html(""+time);
							if (time <= 0) {
								$("#reload .reload").trigger("click");
							}
						}, 1000);
					}
				});
				$("#reload .stop").on("click", function(e){
					e.preventDefault();
					if (interval != null) {
						clearInterval(interval);
						interval = null;
					}
					timer.html("#");
				});
			});
			$("#reload .reload").trigger("click");
		};
		socket.onmessage = function(evt) {
			jsonData = jQuery.parseJSON(evt.data);
			log("WebSocket read: <code>"+jsonData+"</code>");
		};
	}
	else {
		log("Your browser does not support WebSockets.");
		tabs.tabs.forEach(function(val, key) {
				$(key).hide();
				val.hide();
		});
		$(tabs.pages.logpage.header).show();
		tabs.select(tabs.pages.logpage.header);
	}
});
