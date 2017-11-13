//require jQuery
//require helpers.js
$(function() {
	log("init");

	//TODO options & config pages

	//TODO prevent context menu to options (default: do not prevent)
	//prevent context munu cause long tap produces right click on chromiuim
	$(".page").on("contextmenu", function(e) {
		e.preventDefault();
	});

	//TODO to own file (load only on debug)
	//ttc test
	$("#ttcpage div")
	.ttc()
	.on("mousedown mousemove mouseup click", function(e){
		log(xpath(this)+" on "+e.type, {color:"gold"});
		var $span = $($(this).find("span."+e.type)[0]);
		$span.text(parseInt($span.text())+1);
	});
	//ttc test

	var tabs = new Tabs("#header .tab");
	$(tabs.pages.keypage.header).on("select", function(e) {
		log("tabs.pages.keypage.header selected");
		keyinputs.focus();
	});


	//TODO to log.js
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

		var wsProtocol = window.location.protocol.replace("http", "ws");
		log("WebSocket connecting to "+wsProtocol+" ...");
		var socket = new window.WebSocket(wsProtocol+"//"+window.location.host+"/ws");
		socket.onopen = function(evt) {
			log("WebSocket connected");

			//TODO really? this is checked, but others not?
			//TODO check for required components on top
			if (typeof(userConfig) == "object") {
				//TODO to init like keys
				userConfig.socket = socket;
			}

			//TODO to init like keys
			//TODO to it's file, like keys
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


			keyinputs.init(socket);
			modifiers.init(socket);
			keys.init(socket);

		};
		socket.onclose = function(evt) {
			log("WebSocket closed");

			//TODO really? this is checked, but others not?
			//TODO to destroy like keys
			if (typeof(userConfig) == "object") {
				userConfig.socket = null;
			}

			//TODO to destroy like keys
			pad.off("touchtap");
			pad.off("touchdoubletap");
			pad.off("touchmoverelative");
			pad.off("touchscroll");

			left.off("touchdown");
			left.off("touchup");
			left.off("touchdownlock");
			left.off("touchdownunlock");

			right.off("touchdown");
			right.off("touchup");

			keyinputs.destroy();
			modifiers.destroy();
			keys.destroy();

			tabs.tabs.forEach(function(val, key) {
				$(key).hide();
				val.hide();
			});
			$(tabs.pages.reload.header).show();
			if(document.getElementById("logpage") !== null) {
				$(tabs.pages.logpage.header).show();
			}
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
				var url = window.location.protocol+"//"+window.location.host+"/ping";
				$.ajax(url, {
					success: function(){
						window.location.reload();
					},
					error:function(){
						log("Server offline: "+url);
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
				//TODO why is this inside click?
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
			m = jQuery.parseJSON(evt.data);
			log("WebSocket message");
			//log("WebSocket read: <code>"+evt.data+"</code>");
			if (typeof m.type != "string") {
				log("no \"type\" in message", {level:1, color:"red"});
				return;
			}
			log("got \""+m.type+"\": "+JSON.stringify(m.data), {level:1});
			switch (m.type) {
				case "keyinput":
					keyinputs.message(m.data);
					break;
				case "key":
					keys.message(m.data);
					break;
				case "cookieConfig":
					userConfig.message(m.data);
					break;
				default:
					log("unknown \"type\": "+m.type, {level:1, color:"red"});
			}
		};
	}
	else {
		log("Your browser does not support WebSockets", {color:"red"});
		tabs.tabs.forEach(function(val, key) {
				$(key).hide();
				val.hide();
		});
		$("#noWebSocketPage").show();
		if(document.getElementById("logpage") !== null) {
			$(tabs.pages.logpage.header).show();
		}
	}
});
