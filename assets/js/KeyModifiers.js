//require helpers.js
var modifiers = {
	socket: null,
	modifier: {},
	focus: null
};

modifiers.init = function(socket) {
	log("init modifiers");
	this.socket = socket;

	$("button.modifier").each(function() {
		modifiers.add(this);
	});
};

modifiers.add = function(elm) {
	log("modifiers.add("+xpath(elm)+")");
	data = $(elm).data();
	if (typeof(data.name) != "string" || data.name=="") {
		log("no data-name for modifier", {level:1});
		return;
	}
	if (typeof(this.modifier[data.name]) == "undefined") {
		this.modifier[data.name] = new modifier(this.socket, data.name);
	}
	this.modifier[data.name].add(elm);
};

modifiers.destroy = function() {
	log("modifiers.destroy()");
	for (var name in this.modifier) {
		this.modifier[name].destroy();
		delete this.modifier[name];
	}
};

modifiers.message = function(msg) {
	log("modifiers message");
	if (typeof(this.modifier[msg.name]) == "undefined") {
		log("unknown name: "+msg.name, {level:1, color: "red"});
		return;
	}
	this.modifier[msg.name].message(msg);
};

modifiers.relase = function() {
	log("modifiers relase");
	for (var name in this.modifier) {
		this.modifier[name].relase();
	}
};

function modifier(socket, name) {
	log("new modifier: "+name, {color: "pink"});
	this.socket = socket;
	this.name = name;
	this.down = false;
	this.elements = [];
	this.onclick = function(e) {
		e.preventDefault();
		log("modifier "+this.name+" on click", {color:"lightgreen"});
		this.socket.send(JSON.stringify({
			type: "modifier",
			data: {
				name: this.name,
				down: !this.down
			}
		}));
	};
	this.add = function(elm) {
		log("modifier "+this.name+" add element: "+xpath(elm));
		$(elm).on("click.modifier", this.onclick.bind(this));
		this.elements.push(elm);
	};
	this.destroy = function() {
		log("modifier "+this.name+" destroy");
		$(this.elements).each(function(){
			log("off click: "+xpath(this), {level:1});
			$(this).off("click.modifier");
		});
	};
	this.message = function(msg) {
		log("modifier "+this.name+" message: "+JSON.stringify(msg));
		if (typeof(msg.down) != "boolean") {
			log("message down is not boolean: "+typeof(msg.down), {level:1, color: "red"});
		}
		this.down = msg.down;
		if (this.down) {
			$(this.elements).addClass("down");
			return;
		}
		$(this.elements).removeClass("down");
	};
	this.relase = function() {
		log("modifier "+this.name+" relase");
		if (this.down) {
			this.socket.send(JSON.stringify({
				type: "modifier",
				data: {
					name: this.name,
					down: !this.down
				}
			}));
		}
	};
}
