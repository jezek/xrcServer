function xpath(el) {
  if (typeof el == "string") return document.evaluate(el, document, null, 0, null);
  if (!el || el.nodeType != 1) return '';
  if (el.id) return "//*[@id='" + el.id + "']";
  var sames = [].filter.call(el.parentNode.children, function (x) { return x.tagName == el.tagName; });
  return xpath(el.parentNode) + '/' + el.tagName.toLowerCase() + (sames.length > 1 ? '['+([].indexOf.call(sames, el)+1)+']' : '');
}

function log(t, opt) {
	opt = opt || {};
	var elm = $('<p/>');
	if (typeof opt.color === "string") {
		elm.css("color", opt.color);
	}
	if (typeof opt.level === "number") {
		elm.css("margin-left", ""+opt.level+"em");
	}
	elm.html(t);
	$("#log").append(elm);
}

if (!Object.keys) {
	Object.keys = function (obj) {
		var keys = [],
			k;
		for (k in obj) {
			if (Object.prototype.hasOwnProperty.call(obj, k)) {
				keys.push(k);
			}
		}
		return keys;
	};
}


// register window.touchtoclick
// using anonymous function to prevent others to create the same object
var touchtoclick = function() {
	var elements = new Map();

	var canListen = function(e) {
		return e && typeof(e.addEventListener) === "function" && typeof(e.removeEventListener) === "function";
	};

	var simulateMouseEvent = function(target, type, touch) {
		log("simulateMouseEvent: "+type, {level:1});
		//TODO use average for multitouch for screenX/Y, clientX/Y
		var simulated = new MouseEvent(type, {
			"bubbles":       true,
			"cancelable":    true,
			"view":          window,
			"detail":        1,
			"screenX":       touch.screenX,
			"screenY":       touch.screenY,
			"clientX":       touch.clientX,
			"clientY":       touch.clientY,
			"ctrlKey":       false,
			"altKey":        false,
			"shiftKey":      false,
			"metaKey":       false,
			"button":        0,
			"relatedTarget": null,
		});

		log("dispatch simulated: "+simulated.type, {level:1});
		target.dispatchEvent(simulated);
	};

	// https://developer.mozilla.org/en-US/docs/Web/API/EventTarget/addEventListener#Safely_detecting_option_support
	var captureSupported = false;
	try {
		var tmp = Object.defineProperty({}, "capture", {
			get: function() { captureSupported = true; }
		});
		window.addEventListener("test", null, tmp);
	} catch(err) {}
	var capture = false;
	var eventListenerOptions = captureSupported ? { capture: capture } : capture;

	var stopPropagation = function(e) {
		e = e || window.event;
		e.cancelBubble = true;
		if (e.stopPropagation) {e.stopPropagation();}
	};

	var Touchtoclick = function(elm) {
		//TODO do not send first mousedown, if touchmove occured within some time 
		var moved = false;
		var touch = null;
		var ontouchstart = function(e){
			log("ttc: touchstart: "+xpath(elm), {color: "lime"});
			stopPropagation(e);
			if (touch != null) {
				return;
			}
			if (e.changedTouches.length < 1) {
				return;
			}
			moved = false;
			touch = e.changedTouches[0];
			simulateMouseEvent(e.target, "mousedown", touch);
			e.preventDefault();
		}.bind(this);
		var ontouchmove = function(e){
			log("ttc: touchmove: "+xpath(elm), {color: "limegreen"});
			stopPropagation(e);
			if (touch === null) {
				return;
			}
			var changed = null;
			for (var i = 0; i < e.changedTouches.length; i++) {
				if (e.changedTouches[i].identifier === touch.identifier) {
					changed = e.changedTouches[i];
					break;
				}
			}
			if (changed === null) {
				return;
			}
			if (!moved) {
				var dx = changed.screenX - touch.screenX;
				var dy = changed.screenY - touch.screenY;
				//TODO to options
				if (dx*dx + dy*dy < 200) {
					return;
				}
			}
			moved = true;
			touch = changed;
			simulateMouseEvent(e.target, "mousemove", touch);
			e.preventDefault();
		}.bind(this);
		var ontouchend = function(e){
			log("ttc: touchend: "+xpath(elm), {color: "lightgreen"});
			stopPropagation(e);
			//TODO duplicatecode with ontouchmove
			if (touch === null) {
				return;
			}
			var changed = null;
			for (var i = 0; i < e.changedTouches.length; i++) {
				if (e.changedTouches[i].identifier === touch.identifier) {
					changed = e.changedTouches[i];
					break;
				}
			}
			if (changed === null) {
				return;
			}
			simulateMouseEvent(e.target, "mouseup", touch);
			if (!moved) {
				simulateMouseEvent(e.target, "click", touch);
			}
			touch = null;
			moved = false;
		}.bind(this);

		elm.addEventListener("touchstart", ontouchstart, eventListenerOptions);
		elm.addEventListener("touchmove", ontouchmove, eventListenerOptions);
		elm.addEventListener("touchend", ontouchend, eventListenerOptions);

		this.remove = function() {
			elm.removeEventListener("touchstart", ontouchstart, eventListenerOptions);
			elm.removeEventListener("touchmove", ontouchmove, eventListenerOptions);
			elm.removeEventListener("touchend", ontouchend, eventListenerOptions);
		};
	};

	this.add = function (element, options) {
		if (!canListen(element)) {
			console.warn("touchtoclick.add: element is not an event listener: ", element);
			return;
		}
		this.remove(element);
		elements.set(element, new Touchtoclick(element));
		return this;
	};

	this.remove = function(element) {
		if (!canListen(element)) {
			console.warn("touchtoclick.remove: element is not an event listener: ", element);
			return;
		}
		if (!elements.has(element)) {
			return;
		}
		elements.get(element).remove();
		elements.delete(element);
		return this;
	};

	//TODO just for debug, remove if all is working
	this.elements = function() {
		return elements;
	};

	return this;
}.bind({})();

var ttc = function(element, options) {
	return window.touchtoclick.add(element, options);
};

// extend jQuery to touchtoclick
if (typeof(window.jQuery) != "undefined" && typeof(window.jQuery.fn.extend) == "function") {
	window.jQuery.fn.extend({
		ttc: function(options) {
			return this.each(function() {
				ttc(this, options);
			});
		}
	});
}
