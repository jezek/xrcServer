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

$(function() {
	$("input.keyinput").each(function() {
		this.keyinput = new KeyInput(this);
	});
});
