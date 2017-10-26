//require jQuery
//require helpers.js
var keyinputs = {
	socket: null,
	keyinput: {},
	focusingValue: null
};

Object.defineProperty(keyinputs, 'focusing', {
	get: function() {
		return this.focusingValue;
	},
	set: function(value) {
		if (this.focusingValue === value) {
			return;
		}
		if (typeof(userConfig) == "object" && typeof(userConfig.update) == "function") {
			userConfig.update({
				keyinputsFocusing: value || ""
			});
			log("keyinputs.focusing: updating in userConfig: value");
		}
		this.focusingValue = value;
	}
});

keyinputs.init = function(socket) {
	log("init keyinputs");
	this.socket = socket;

	$("input.keyinput").each(function() {
		keyinputs.add(this);
	});

	if (typeof(userConfig) == "object" && typeof(userConfig.data) == "function") {
		var data = userConfig.data();

		if (typeof(data.keyinputsFocusing) == "undefined" || data.keyinputsFocusing=="") {
			data.keyinputsFocusing=null;
		}

		log("keyinputs.init: got focusing from userConfig");
		this.focusingValue=data.keyinputsFocusing;
		this.focus();
	}
};

keyinputs.add = function(elm) {
	log("keyinputs add "+xpath(elm));
	data = $(elm).data();
	if (typeof(data.name) != "string" || data.name=="") {
		data.name = Object.keys(this.keyinput).length + 1;
	}
	if (typeof(this.keyinput[data.name]) != "undefined") {
		log("duplicate name: "+data.name);
		return;
	}
	this.keyinput[data.name] = new keyinput(this.socket, elm, data.name);
};

keyinputs.destroy = function() {
	log("keyinputs.destroy()");
	for (var name in this.keyinput) {
		this.keyinput[name].destroy();
		delete this.keyinput[name];
	}
};

keyinputs.message = function(msg) {
	log("keyinputs message");
	if (typeof(this.keyinput[msg.sender]) == "undefined") {
		log("unknown sender: "+msg.sender, {level:1, color: "red"});
	}
	this.keyinput[msg.sender].message(msg);
};


keyinputs.focus = function() {
	log("keyinputs.focus", {color: "pink"});
	

	if (this.focusing == null) {
		log("no focusing", {level:1, color: "pink"});
		return;
	}

	if (typeof(this.keyinput[this.focusing]) == "undefined") {
		log("bad focusing", {level:1, color: "pink"});
		this.focusing=null;
		return;
	}

	log("focusing: "+this.focusing, {level:1, color: "pink"});
	$(this.keyinput[this.focusing].elm).focus();
	log("focused: "+this.focusing, {level:1, color: "pink"});
};

function keyinput(socket, elm, name, opt) {
	log("new keyinput: "+name, {color: "pink"});
	this.socket = socket;
	this.elm = elm;
	this.name = name;

	this.options = Object.assign({
		autorefocus: true
	}, $(this.elm).data(), opt);

	this.lastValue = "";
	this.wasInput = false;

	$(this.elm).on("keydown.keyinput", function(e) {
		log("keydown", {color: "cyan"});
		log("e.key: "+e.key, {level: 1});
		log("e.keyCode: "+e.keyCode, {level: 1});
		log("this.lastValue: \""+this.lastValue+"\"", {level: 1});
		log("$(this.elm).val(): \""+$(this.elm).val()+"\"", {level: 1});

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
		$(this.elm).trigger($.Event("keyinput", {
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

	$(this.elm).on("input.keyinput", function(e) {
		log("input", {color: "lightblue"});
		log("this.lastValue: \""+this.lastValue+"\"", {level: 1});
		log("$(this.elm).val(): \""+$(this.elm).val()+"\"", {level: 1});

		this.wasInput = true;

		diff = this.textDifference(this.lastValue, $(this.elm).val());
		replaced = $(this.elm).val().indexOf(this.lastValue) != 0 && this.lastValue.indexOf($(this.elm).val()) != 0;
		log("replaced: "+replaced, {level: 1});
		this.lastValue = $(this.elm).val();
		$(this.elm).val("");
		if (diff === "") {
			this.lastValue="";
			return;
		}

		$(this.elm).trigger($.Event("keyinput", {
			text: diff
		}));
		if (diff === " " || replaced) {
			this.lastValue="";
			return;
		}
	}.bind(this));

	$(this.elm).on("keyup.keyinput", function(e) {
		log("keypup", {color: "lightblue"});
		log("e.key: "+e.key, {level: 1});
		log("e.keyCode: "+e.keyCode, {level: 1});
		log("this.lastValue: \""+this.lastValue+"\"", {level: 1});
		log("$(this.elm).val(): \""+$(this.elm).val()+"\"", {level: 1});
		log("this.wasInput: "+this.wasInput, {level: 1});

		diff = this.textDifference(this.lastValue, $(this.elm).val());

		if (this.wasInput) {
			if (diff == " ") {
				this.lastValue="";
			}
			return;
		}

		autocorrect = this.lastValue.length>1 && $(this.elm).val()=="";

		this.lastValue = $(this.elm).val();
		$(this.elm).val("");

		if (diff === "" || autocorrect) {
			return;
		}

		$(this.elm).trigger($.Event("keyinput", {
			text: diff
		}));

	}.bind(this));

	$(this.elm).on("keyinput", function(e) {
		log("keyinput", {color: "blue"});
		e.text = e.text.replace("%", "%%");

		if (typeof(modifiers) != "undefined") {
			e.text = modifiers.apply(e.text);
		}

		log("e.text: \"<code>"+e.text+"</code>\"", {level: 1});
		this.socket.send(JSON.stringify({
			type: "keyinput",
			data: {
				text: e.text,
				sender: this.name
			}
		}));
	}.bind(this));

	this.destroy = function() {
		log("keyinput "+this.name+" destroy");
			$(this.elm)
				.off("keyup.keyinput")
				.off("input.keyinput")
				.off("keydown.keyinput")
				.off("focus.keyinput")
				.off("focusout.keyinput")
				.off("keyinput");
	};

	this.message = function(msg) {
		pos = $(this.elm).position();
		$(this.elm)
			.finish()
			.css({borderColor:"green"})
			.attr("placeholder", m.data.text)
		  .animate({borderColor:"initial"}, 400, "swing", function() {
				$(this)
					.attr("placeholder", "")
					.css({borderColor:"initial"});
			});
	};

	this.placeholder = null;

	$(this.elm).on("focus.keyinput", function(e) {
		log("keyinput on focus: "+this.name, {color:"yellow"});
		this.placeholder = $(this.elm).attr("placeholder");
		$(this.elm).attr("placeholder","");
	}.bind(this));

	$(this.elm).on("focusout.keyinput", function(e) {
		log("keyinput on focusout: "+this.name, {color:"brown"});
		$(this.elm).finish().attr("placeholder",this.placeholder);
		this.placeholder = null;
	}.bind(this));

	$(this.elm).on("click", function(e) {
		e.preventDefault();
		log("keyinput on click: "+this.name);
		if (keyinputs.focusing == this.name) {
			log("unfocus", {level:1});
			keyinputs.focusing = null;
			$(this.elm).blur();
			return;
		}

		log("focus", {level:1});
		keyinputs.focusing = this.name;

	}.bind(this));

}
