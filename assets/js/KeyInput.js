//require helpers.js
var keyinputs = {
	socket: null,
	keyinput: {},
	focused: null
};

keyinputs.init = function(socket) {
	log("init keyinputs");
	this.socket = socket;

	$("input.keyinput").each(function() {
		keyinputs.add(this);
	});
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

keyinputs.focus = function(name, opt) {
	log("keyinputs focus: "+name, {color:"orange"});
	log("opt: "+JSON.stringify(opt), {level:1});
	if (typeof(this.keyinput[name]) == "undefined") {
		log("unknown name", {level:1});
		return;
	}
	if (this.focused == name) {
		log("allreay focused", {level:1});
		return;
	}
	if (this.focused != null) {
		this.keyinput[this.focused].unfocus();
	}
	this.focused = name;
	this.keyinput[name].focus(opt);
};

keyinputs.unfocus = function() {
	log("keyinputs unfocus", {color:"brown"});
	if (this.focused == null) {
		log("allready unfocused", {level:1});
		return;
	}
	this.keyinput[this.focused].unfocus();
	this.focused = null;
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
		log("e.text: \"<code>"+e.text+"</code>\"", {level: 1});
		var codes="";
		for (i=0; i<e.text.length; i++) {
			if (codes != "") {
				codes +=",";
			}
			codes += ""+e.text.charCodeAt(i);
		}
		log("e.text: codes: "+codes, {level: 1});
		this.socket.send(JSON.stringify({
			type: "keyinput",
			data: {
				text: e.text.replace("%", "%%"),
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
		//TODO use placeholder?
		$("<div/>").css({
			position: "absolute",
			top: pos.top+"px",
			left: pos.left+"px",
			width: $(this.elm).outerWidth()+"px",
			height: $(this.elm).outerHeight()+"px",
			display: "flex",
			"justify-content": "center",
			"align-content": "center",
			"align-items": "center"
		}).text(m.data.text)
			.appendTo($(this.elm).parent())
			.animate({opacity:0}, 400, "swing", function() {
				$(this).remove();
			});
		if (typeof(modifiers) != "undefined") {
			modifiers.relase();
		}
	};

	this.placeholder = null;

	this.focus = function(opt) {
		log("keyinput focus: "+this.name, {color:"orange"});
		log("opt: "+JSON.stringify(opt), {level:1});
		if (typeof(opt) != "undefined" && typeof(opt.clear) == "boolean") {
			this.lastValue = "";
		}
		$(this.elm)
			.off("focusout.keyinput")
			.on("focusout.keyinput", function(e) {
				//TODO use other approach, this breaks things if url bar in browser is clicked
				log("keyinput on focusout: "+this.name, {color:"yellow"});
				if ($(this.elm).is(":visible") == false) {
					log("element is hidden", {level:1});
					keyinputs.unfocus();
					return;
				}
				keyinputs.unfocus();
				if (this.options.autorefocus == true) {
					log("autorefocus", {level:1});
					$(this.elm).focus();
					return;
				}
			}.bind(this));
		if (typeof(opt) == "undefined" || typeof(opt.onfocus) == "undefined") {
			$(this.elm).focus();
		}
	};

	this.unfocus = function() {
		log("keyinput unfocus: "+this.name, {color:"brown"});
		$(this.elm).off("focusout.keyinput");
		$(this.elm).attr("placeholder",this.placeholder);
		this.placeholder = null;
	};

	$(this.elm).on("focus.keyinput", function(e) {
		log("keyinput on focus: "+this.name, {color:"yellow"});
		this.placeholder = $(this.elm).attr("placeholder");
		$(this.elm).attr("placeholder","");
		keyinputs.focus(this.name, {onfocus: true});
	}.bind(this));

}
