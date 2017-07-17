function TouchPad(elm) {

	this.elm = $(elm);
	this.touches = new Map();
	
	//log("this.elm on touchstart");
	this.elm.on("touchstart", function(e) {
		//log("touchstart");
		//log("this: "+JSON.stringify(this));
		//log("e: "+JSON.stringify(e));
		e.preventDefault();
		for (var i = 0; i < e.changedTouches.length; i++) {
			var t = e.changedTouches[i];
			switch (this.touches.size) {
				case 0:
				case 1:
					this.touches.set(t.identifier, {
						last: t,
						sizeCreated: this.touches.size
					});
					//log("touches.size "+this.touches.size);
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
				for (i = 0; i < e.changedTouches.length; i++) {
					t = e.changedTouches[i];
					if (!this.touches.has(t.identifier)) {
						continue;
					}
					touch = this.touches.get(t.identifier);
					t.dx = t.screenX-touch.last.screenX;
					t.dy = t.screenY-touch.last.screenY;
					if (touch.moved == undefined && ((t.dx*t.dx)+(t.dy*t.dy)) < 200) {
						break;
					}
					touch.moved = true;
					touch.preventClick = true;
					//log("trigger: touchmoverelative");;
					this.elm.trigger("touchmoverelative", [t]);
					touch.last = t;
					break;
				}
				break;
			case 2:
				// 2 finger move
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
				this.touches.entries().forEach(function() {
					this.moved = true;
					this.preventClick = true;
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
		//log("touches.size "+this.touches.size);
		e.preventDefault();
		for (var i = 0; i < e.changedTouches.length; i++) {
			var t = e.changedTouches[i];
			if (!this.touches.has(t.identifier)) {
				continue;
			}
			var touch = this.touches.get(t.identifier);
			switch (this.touches.size) {
				case 1:
					if (touch.preventClick == true) {
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
					this.touches.entries().forEach(function() {
						this.rightClick=true;
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
