function TouchButton(elm) {

	this.elm = $(elm);
	this.fingers = new Map();

	this.elm.on("touchstart", function(e) {
		e.preventDefault();
		//log("this.elm on: touchstart");
		for (var i = 0; i < e.changedTouches.length; i++) {
			var t = e.changedTouches[i];
			switch (this.fingers.size) {
				case 0:
					this.fingers.set(t.identifier, t);
					if (!this.elm.hasClass("locked")) {
						log("left down");
						this.elm.addClass("down");
					} else {
						this.elm.removeClass("locked");
					}
					break;
				default:
					if (!this.elm.hasClass("locked")) {
						this.elm.addClass("locked");
					}
					return;
			}
		}
	}.bind(this));
	this.elm.on("touchend", function(e) {
		e.preventDefault();
		//log("this.elm on: touchstop");
		for (var i = 0; i < e.changedTouches.length; i++) {
			var t = e.changedTouches[i];
			if (!this.fingers.has(t.identifier)) {
				continue;
			}
			if (!this.elm.hasClass("locked")) {
				log("left up");
				this.elm.removeClass("down");
			}
			this.fingers.delete(t.identifier);
			return;
		}
	}.bind(this));
}
