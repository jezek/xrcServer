//require jQuery
//require helpers.js
//TODO? option on how to use modifiers (direct send or send with key)
//TODO? in future subscribe for modifiers watch on server side (yo can see if key is pressed)???
var modifiers = {
	modifier: {},
};

modifiers.init = function() {
	log("init modifiers");

	var isTtc = (typeof(window.ttc) === "function");
	if (!isTtc) {
		log("no touchtoclick", {level:1});
	}

	$("button.modifier").each(function() {
		if (isTtc) {
			window.ttc(this);
		}
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
		this.modifier[data.name] = new modifier(data.name);
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

modifiers.relaseIfNotLocked = function() {
	log("modifiers relase not locked");
	for (var name in this.modifier) {
		this.modifier[name].relaseIfNotLocked();
	}
};

modifiers.getAllDown = function(relase) {
	log("get down modifiers");
	relase = relase || false;
	if (typeof(relase) !== "boolean") {
		log("relase param is not boolean", {color: "red", level:1});
		return;
	}

	var res = [];
	for (var name in this.modifier) {
		if (this.modifier[name].down) {
			res.push(name);
			if (relase) {
				this.modifier[name].relaseIfNotLocked();
			}
		}
	}
	return res;
};

modifiers.apply = function(text) {
	log("apply modifiers to: "+text);
	if (typeof(text) !== "string") {
		log("text param is not string", {color: "red", level:1});
		return text;
	}
	var prefix = "";
	var suffix = "";
	var mods = this.getAllDown(true);
	if (Array.isArray(mods)) {
		mods.forEach(function(mod) {
			prefix = prefix+"%+%\""+mod+"\"";
			suffix = suffix+"%-%\""+mod+"\"";
		});
	}
	return prefix + text + suffix;
};

function modifier(name) {
	log("new modifier: "+name, {color: "pink"});
	this.name = name;
	this.down = false;
	this.locked = false;
	this.elements = [];

	this.options = {
		lockDelay: 1000
	};

	var lockTimer = null;

	this.updateElements = function() {
		if (this.down) {
			$(this.elements).addClass("down");
		} else {
			$(this.elements).removeClass("down");
		}
		if (this.locked) {
			$(this.elements).addClass("locked");
		} else {
			$(this.elements).removeClass("locked");
		}
	};

	var ondown = function(e) {
		log("modifier "+this.name+" on down", {color:"lightgreen"});
		if (lockTimer !== null) {
			clearTimeout(lockTimer);
			lockTimer = null;
		}
		if (this.locked) {
			this.locked = false;
			this.down = false;
		} else {
			this.down = !this.down;
		}
		this.updateElements();
		if (this.options.lockDelay > 0) {
			lockTimer = setTimeout(function() {
				//TODO? on lock press actually mod down and dont send with keys as pref+suff
				log("modifier "+this.name+" lock callback", {color:"lightgreen"});
				this.locked = true;
				this.down = true;
				lockTimer = null;
				this.updateElements();
			}.bind(this), this.options.lockDelay);
		}
		e.preventDefault();
	}.bind(this);

	var onup = function(e) {
		log("modifier "+this.name+" on up", {color:"lightgreen"});
		if (lockTimer === null) {
			log("no lock timer", {color:"lightgreen", level:1});
			return;
		}
		clearTimeout(lockTimer);
		lockTimer = null;
		e.preventDefault();
	}.bind(this);

	this.add = function(elm) {
		log("modifier "+this.name+" add element: "+xpath(elm));
		$(elm)
		  .on("mousedown.modifier", ondown)
		  .on("mouseup.modifier", onup)
		  .on("mouseleave.modifier", onup);
		this.elements.push(elm);
		this.updateElements();
	};
	this.destroy = function() {
		log("modifier "+this.name+" destroy");
		$(this.elements).each(function(){
			log("off click: "+xpath(this), {level:1});
			$(this)
				.off("mousedown.modifier")
				.off("mouseup.modifier")
				.off("mouseleave.modifier");
		});
		this.down = false;
		this.locked = false;
		this.updateElements();
		this.elements = [];
	};
	this.relaseIfNotLocked = function() {
		log("modifier "+this.name+" relase");
		if (!this.locked) {
			this.down = false;
		}
		this.updateElements();
	};
}
