//require jQuery
//require helpers.js, KeyModifiers.js
//uses touchtoclick
var keys = {
	socket: null,
	key: {}
};

//TODO? subscribe for mapping change, diferentiate between mapped & unmapped keys
//TODO? in the future map selected unmapped keys
//TODO? in future subscribe for keys watch on server side (you can see if key is pressed)???
keys.init = function(socket) {
	log("init keys");
	this.socket = socket;

	var isTtc = (typeof(window.ttc) === "function");
	if (!isTtc) {
		log("no touchtoclick", {level:1});
	}

	$("button.key").each(function() {
		if (isTtc) {
			window.ttc(this);
		}
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
		this.key[data.name] = new key(data.name);
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


keys.send = function(name) {
	log("keys send: "+name, {color: "violet"});
	if (typeof(this.key[name]) === "undefined") {
		log("unknown name", {color: "red", level:1});
		return;
	}
	var text = "%\""+name+"\"";

	if (typeof(modifiers) != "undefined") {
		text = modifiers.apply(text);
	}

	log("text: \"<code>"+text+"</code>\"", {color: "violet", level: 1});
	this.socket.send(JSON.stringify({
		type: "key",
		data: {
			text: text,
			sender: name
		}
	}));
};

keys.message = function(msg) {
	log("keys message");
	if (typeof(this.key[msg.sender]) == "undefined") {
		log("unknown name: "+msg.sender, {level:1, color: "red"});
		return;
	}
	this.key[msg.sender].message(msg);
};

function key(name) {
	log("new key: "+name, {color: "pink"});
	this.name = name;
	this.down = false;
	this.elements = [];

	this.updateElements = function() {
		if (this.down) {
			$(this.elements).addClass("down");
		} else {
			$(this.elements).removeClass("down");
		}
	};

	this.ondown = function(e) {
		log("key "+this.name+" on down", {color:"gold"});
		if (this.down) {
			log("allready down", {level:1});
			return;
		}
		this.down = true;
		this.updateElements();
		keys.send(this.name);
	};

	this.message = function(msg) {
		log("key "+this.name+" message: "+JSON.stringify(msg));
		this.down = false;
		this.updateElements();
	};

	this.add = function(elm) {
		log("key "+this.name+" add element: "+xpath(elm));
		$(elm)
			.on("mousedown.key", this.ondown.bind(this));
		this.elements.push(elm);
	};

	this.destroy = function() {
		log("key "+this.name+" destroy");
		$(this.elements).each(function(){
			log("off: "+xpath(this), {level:1});
			$(this)
				.off("mousedown.key");
		});
		this.down = false;
		this.updateElements();
		this.elements = [];
	};
}
