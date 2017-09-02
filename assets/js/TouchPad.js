function TouchPad(elm, opt) {
	this.elm = $(elm);

	this.options = Object.assign({
		tap_enabled: true,
		first_move_distance: 200
	}, this.elm.data(), opt);

	this.touches = new Map();

	this.elm.on("touchstart", function(e) {
		log("touchpad: touchstart");
		e.preventDefault();
		for (var i = 0; i < e.changedTouches.length; i++) {
			var t = e.changedTouches[i];
			switch (this.touches.size) {
				case 0:
				case 1:
					this.touches.set(t.identifier, {
						last: t,
						sizeCreated: this.touches.size,
						moved: false,
						preventClick: !this.options.tap_enabled
					});
					//log(t.identifier+": set touch "+this.touches.size);
					break;
				default:
					return;
			}
		}
	}.bind(this));
	this.elm.on("touchmove", function(e) {
		//log("touchmove");
		e.preventDefault();
		var i, t, touch;
		switch (this.touches.size) {
			case 1:
				// 1 finger move
				//log("1 finger move");
				for (i = 0; i < e.changedTouches.length; i++) {
					t = e.changedTouches[i];
					if (!this.touches.has(t.identifier)) {
						continue;
					}
					touch = this.touches.get(t.identifier);
					t.dx = t.screenX-touch.last.screenX;
					t.dy = t.screenY-touch.last.screenY;

					if (touch.moved == false) {
						if (this.options.tap_enabled && (t.dx*t.dx)+(t.dy*t.dy) < this.options.first_move_distance) {
							break;
						}
						//log(t.identifier+": first move");
						touch.moved = true;
						touch.preventClick = true;
					} else {
						//log(t.identifier+": trigger: touchmoverelative");
						this.elm.trigger("touchmoverelative", [t]);
					}
					touch.last = t;
					break;
				}
				break;
			case 2:
				// 2 finger move
				//log("2 finger move");
				var dy = 0;
				for (i = 0; i < e.changedTouches.length; i++) {
					t = e.changedTouches[i];
					if (!this.touches.has(t.identifier)) {
						continue;
					}
					touch = this.touches.get(t.identifier);
					dy += t.screenY - touch.last.screenY;
				}
				if ((dy*dy) < 100) {
					break;
				}
				//log("scroll "+dy);
				for (i = 0; i < e.changedTouches.length; i++) {
					t = e.changedTouches[i];
					if (!this.touches.has(t.identifier)) {
						continue;
					}
					touch = this.touches.get(t.identifier);
					touch.last = t;
				}
				this.touches.forEach(function(val) {
					val.moved = true;
					val.preventClick = true;
				});
				var dir = "up";
				if (dy > 0) {
					dir = "down";
				}
				//log("trigger: touchscroll: "+dir);
				this.elm.trigger("touchscroll", [{dir: dir}]);
				break;
			default:
		}
	}.bind(this));
	this.elm.on("touchend", function(e) {
		//log("touchend");
		//log("this.touches.size "+this.touches.size);
		e.preventDefault();
		for (var i = 0; i < e.changedTouches.length; i++) {
			var t = e.changedTouches[i];
			if (!this.touches.has(t.identifier)) {
				continue;
			}
			var touch = this.touches.get(t.identifier);
			switch (this.touches.size) {
				case 1:
					if (!this.options.tap_enabled || touch.preventClick == true) {
						break;
					}
					if (touch.rightClick == true) {
						//log("trigger: touchdoubletap");
						this.elm.trigger("touchdoubletap");
						break;
					}
					//log("trigger: touchtap");
					this.elm.trigger("touchtap");
					break;
				case 2:
					this.touches.forEach(function(val) {
						val.rightClick=true;
					});
					break;
				default:
			}
			this.touches.delete(t.identifier);
		}
	}.bind(this));
	this.elm.on("touchcancel", function(e) {
		//todo
	}.bind(this));
}
