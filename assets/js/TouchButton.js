//require jQuery
function TouchButton(elm, opt) {
	this.elm = $(elm);

	this.options = Object.assign({
		lockable: false,
		up_down_distance_ms: 50
	}, this.elm.data(), opt);

	this.maxTouches = 0;
	this.locked = false;
	this.up_down_timer = null;

	this.elm.on("touchstart", function(e) {
		e.preventDefault();
		log(this.elm.prop("id")+" on: touchstart");
		//log("e.touches.length: "+e.touches.length);
		//log("this.maxTouches: "+this.maxTouches);
		//log("this.locked: "+this.locked);


		if (this.up_down_timer != null) {
			//log("clearTimeout");
			clearTimeout(this.up_down_timer);
			this.up_down_timer = null;
		} else {
			if (this.maxTouches == 0) {
				if (this.options.lockable == false || !this.locked) {
					//log("this.elm.trigger: touchdown");
					this.elm.trigger("touchdown");
				} else {
					if (e.touches.length == 1) {
						this.locked=false;
						//log("this.elm.trigger: touchdownunlock");
						this.elm.trigger("touchdownunlock");
					}
				}
			} 
		}

		if (this.options.lockable && !this.locked) {
			if (e.touches.length > 1) {
				this.locked=true;
				//log("this.elm.trigger: touchdownlock");
				this.elm.trigger("touchdownlock");
			}
		}

		if (this.maxTouches < e.touches.length) {
			this.maxTouches = e.touches.length;
		}

	}.bind(this));
	this.elm.on("touchend", function(e) {
		e.preventDefault();
		//log(this.elm.prop("id")+" on: touchend");
		//log("e.touches.length: "+e.touches.length);
		//log("this.maxTouches: "+this.maxTouches);
		//log("this.locked: "+this.locked);

		if (this.up_down_timer != null) {
			//log("clearTimeout");
			clearTimeout(this.up_down_timer);
			this.up_down_timer = null;
		}

		if (e.touches.length == 0) {
			//log("setTimeout");
			this.up_down_timer = setTimeout(function() {
				if (this.options.lockable == false || !this.locked) {
					//log("this.elm.trigger: touchup");
					this.elm.trigger("touchup");
				}
				this.maxTouches = 0;
				this.up_down_timer = null;
			}.bind(this), this.options.up_down_distance_ms);
		}
	}.bind(this));
}
