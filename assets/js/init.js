function log(t, opt) {
	opt = opt || {};
	var elm = $('<p/>');
	if (typeof opt.color === "string") {
		elm.css("color", opt.color);
	}
	if (typeof opt.level === "number") {
		elm.css("margin-left", ""+opt.level+"em");
	}
	elm.html(t);
	$("#log").append(elm);
}

$(function() {
	log("init");
	var tabs = new Tabs("#header .tab");

	$(tabs.pages.keypage.header).on("select", function(e) {
		//log("keypage selected");
		$("#keypage input.keyinput").focus();
	});

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
		var keyinput = $("#keypage input.keyinput");

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


			// keyinput

			keyinput.on("keyinput", function(e) {
				log("keyinput", {color: "red"});
				log("e.text: \"<code>"+e.text+"</code>\"", {level: 1});
				var codes="";
				for (i=0; i<e.text.length; i++) {
					if (codes != "") {
						codes +=",";
					}
					codes += ""+e.text.charCodeAt(i);
				}
				log("e.text: codes: "+codes, {level: 1});
				socket.send(JSON.stringify({
					type: "key",
					data: {
						text: e.text.replace("%", "%%"),
						sender: "#"+$(this).attr("id")
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

			right.off("touchdown");
			right.off("touchup");

			keyinput.off("keyinput");

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
			log("WebSocket read: <code>"+evt.data+"</code>");
			if (typeof m.type != "string") {
				log("no \"type\"");
				return;
			}
			switch (m.type) {
				case "key-confirm":
					log("got \"key-confirm\": "+m.data.text);
					target = $(m.data.sender);
					pos = target.offset();
					$("<div/>").css({
						position: "fixed",
						top: (3+pos.top)+"px",
						left: (3+pos.left)+"px"
					}).text(m.data.text)
					.appendTo("body")
					.animate({opacity:0, top:"-=30"}, 400, "swing", function() {
						$(this).remove();
						log("key-confirm animation end");
					});

					break;
				default:
					log("unknown \"type\": "+d.type);
			}
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
