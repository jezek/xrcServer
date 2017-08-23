//require helpers.js, KeyModifiers.js
var keys = {
	socket: null,
	key: {}
};

keys.init = function(socket) {
	log("init keys");
	this.socket = socket;

	$(".key").each(function() {
		keys.add(this);
	});
};

keys.add = function(elm) {
	log("keys.add("+xpath(elm)+")");
	data = $(elm).data();
	if (typeof(data.name) != "string" || data.name=="") {
		log("no data-name for key", {level:1});
		return;
	}
	if (typeof(this.key[data.name]) == "undefined") {
		this.key[data.name] = new key(this.socket, data.name);
	}
	this.key[data.name].add(elm);
};

keys.destroy = function() {
	log("keys.destroy()");
	for (var name in this.key) {
		this.key[name].destroy();
		delete this.key[name];
	}
};

keys.message = function(msg) {
	log("keys message");
	if (typeof(this.key[msg.name]) == "undefined") {
		log("unknown name: "+msg.name, {level:1, color: "red"});
	}
	this.key[msg.name].message(msg);
};

function key(socket, name) {
	log("new key: "+name, {color: "pink"});
	this.socket = socket;
	this.name = name;
	this.elements = [];
	this.down = false;
	this.sendKey = function(keysym, down) {
		this.socket.send(JSON.stringify({
			type: "key",
			data: {
				name: this.name,
				down: down
			}
		}));
	};
	this.ondown = function(e) {
		e.preventDefault();
		log("key "+this.name+" on down", {color:"lightgreen"});
		this.sendKey(this.name, true);
	};
	this.onup = function(e) {
		e.preventDefault();
		log("key "+this.name+" on up", {color:"lightgreen"});
		this.sendKey(this.name, false);

		if (typeof(modifiers) != "undefined") {
			if (modifiers.focus != null) {
				//TODO use keyInput objet to handle focus
				//can't pass additional params to focus event
				//use element params
				modifiers.focus.from = "key";
				$(modifiers.focus).focus();
			}
		}
	};
	this.add = function(elm) {
		log("key "+this.name+" add element: "+xpath(elm));
		$(elm)
			.on("touchdown.key", this.ondown.bind(this))
			.on("touchup.key", this.onup.bind(this))
			.on("mousedown.key", this.ondown.bind(this))
			.on("mouseup.key", this.onup.bind(this));
		this.elements.push(elm);
	};
	this.destroy = function() {
		log("key "+this.name+" destroy");
		$(this.elements).each(function(){
			log("off: "+xpath(this), {level:1});
			$(this)
				.off("touchdown.key")
				.off("touchup.key")
				.off("mousedown.key")
				.off("mouseup.key");
		});
	};
	this.message = function(msg) {
		log("key "+this.name+" message: "+JSON.stringify(msg));
		if (typeof(msg.down) != "boolean") {
			log("message down is not boolean: "+typeof(msg.down), {level:1, color: "red"});
		}
		this.down = msg.down;
		if (this.down) {
			$(this.elements).addClass("down");
			return;
		}
		$(this.elements).removeClass("down");
		if (typeof(modifiers) != "undefined") {
			modifiers.relase();
		}
	};
}
