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

			function KeyInput(elm, opt) {
				this.elm = $(elm);

				this.options = Object.assign({
				}, this.elm.data(), opt);

				this.lastValue = "";
				this.wasInput = false;

				this.elm.on("focus", function(e) {
					this.lastValue = "";
				}.bind(this));

				this.elm.on("keydown", function(e) {
					log("keydown", {color: "cyan"});
					log("e.key: "+e.key, {level: 1});
					log("e.keyCode: "+e.keyCode, {level: 1});
					log("this.lastValue: \""+this.lastValue+"\"", {level: 1});
					log("this.elm.val(): \""+this.elm.val()+"\"", {level: 1});

					if (e.keyCode == 229) { //Process
						this.wasInput = false;
						return;
					}

					this.wasInput = true;
					e.preventDefault();
					//log("keydown: char: "+e.char);
					//log("keydown: key: "+e.key);
					//log("keydown: charCode: "+e.charCode);
					//log("keydown: keyCode: "+e.keyCode);
					//log("keydown: repeat: "+e.repeat);
					//log("keydown: ctrlKey: "+e.ctrlKey);
					//log("keydown: altKey: "+e.altKey);
					//log("keydown: shiftKey: "+e.shiftKey);
					//log("keydown: metaKey: "+e.metaKey);


					t = e.key.length > 1 ? String.fromCharCode(e.keyCode) : e.key;
					this.elm.trigger($.Event("keyinput", {
						text: t
					}));
				}.bind(this));

				this.textDifference = function(prev, now) {
					if (prev == now) {
						return "";
					}
					if (now.indexOf(prev) == 0) {
						return now.slice(prev.length);
					} 
					if (prev.indexOf(now) == 0) {
							return String.fromCharCode(8).repeat(prev.length - now.length);
					} 
					return String.fromCharCode(8).repeat(prev.length)+now;
				};

				this.elm.on("input", function(e) {
					log("input", {color: "lightblue"});
					log("this.lastValue: \""+this.lastValue+"\"", {level: 1});
					log("this.elm.val(): \""+this.elm.val()+"\"", {level: 1});

					this.wasInput = true;

					diff = this.textDifference(this.lastValue, this.elm.val());
					replaced = this.elm.val().indexOf(this.lastValue) != 0 && this.lastValue.indexOf(this.elm.val()) != 0;
					log("replaced: "+replaced, {level: 1});
					this.lastValue = this.elm.val();
					this.elm.val("");
					if (diff === "") {
						this.lastValue="";
						return;
					}

					this.elm.trigger($.Event("keyinput", {
						text: diff
					}));
					if (diff === " " || replaced) {
						this.lastValue="";
						return;
					}
				}.bind(this));

				this.elm.on("keyup", function(e) {
					log("keypup", {color: "lightblue"});
					log("e.key: "+e.key, {level: 1});
					log("e.keyCode: "+e.keyCode, {level: 1});
					log("this.lastValue: \""+this.lastValue+"\"", {level: 1});
					log("this.elm.val(): \""+this.elm.val()+"\"", {level: 1});
					log("this.wasInput: "+this.wasInput, {level: 1});
					
					diff = this.textDifference(this.lastValue, this.elm.val());

					if (this.wasInput) {
						if (diff == " ") {
							this.lastValue="";
						}
						return;
					}
					
					autocorrect = this.lastValue.length>1 && this.elm.val()=="";

					this.lastValue = this.elm.val();
					this.elm.val("");

					if (diff === "" || autocorrect) {
						return;
					}

					this.elm.trigger($.Event("keyinput", {
						text: diff
					}));

				}.bind(this));
			}

			// keyinput

			$("input.keyinput").each(function() {
				this.keyinput = new KeyInput(this);
			});

			var keyinput = $("#keypage input.keyinput");
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
						text: e.text,
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
