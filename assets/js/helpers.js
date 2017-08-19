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
