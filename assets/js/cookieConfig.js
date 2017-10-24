function cookieConfig(initConfig) {
	this.socket = null;
	var saved = initConfig;
	var updates = null;
	var sent = null;

	this.update = function(newUpdates) {
		log("config update", {color:"navyblue"});
		if (typeof(newUpdates) != "undefined") {
			if (updates === null) {
				updates={};
			}
			updates = $.extend(true, updates, newUpdates);
		}
		if (this.socket == null || this.updates === null) {
			log("no socket or nothing to update", {level:1});
			return;
		}

		updatesString = JSON.stringify(updates);

		this.socket.send(JSON.stringify({
			type: "cookieConfig",
			data: {
				config: JSON.stringify($.extend(true, saved, updates)),
				updates: updatesString
			}
		}));

		if (sent === null) {
			sent = {};
		}
		sent[updatesString]=updates;
		updates = null;

	};

	var setCookie = function(cookie) {
		if (typeof(cookie) != "object") {
			log("cookie is not an onject", {level:1, color: "red"});
			return;
		}
		if (typeof(cookie.name) != "string") {
			log("cookie.name is not string", {level:1, color: "red"});
			return;
		}
		if (cookie.name.indexOf("=") != -1) {
			log("cookie.name contains '\"' character", {level:1, color: "red"});
			return;
		}
		if (typeof(cookie.value) != "string") {
			log("cookie.value is not string", {level:1, color: "red"});
			return;
		}
		document.cookie = cookie.name+"="+cookie.value;
	};

	this.message = function(msg) {
		log("config message", {color:"cyan"});

		if (sent === null) {
			log("hmm... no message expecting", {level:1, color: "red"});
			return;
		}

		if (typeof(sent[msg.updates]) == "undefined") {
			log("expecting some messages, but not this", {level:1, color: "red"});
			return;
		}

		delete sent[msg.updates];

		if (typeof(msg.error) == "string") {
			log("error returned: "+msg.error, {level:1, color: "red"});
			return;
		}

		if (typeof(msg.config) == "undefined") {
			log("no config", {level:1, color: "red"});
			return;
		}

		if (typeof(msg.cookie) == "undefined") {
			log("no cookie provided", {level:1, color: "red"});
		}
		setCookie(msg.cookie);
		saved = msg.config;
		log("config saved", {level:1});
	};

	this.data = function() {
		return $.extend(true, {}, saved);
	};
}
