function Tabs(elms) {
	this.tabs = new Map();
	this.pages = {};
	this.selected = null;

	this.select = function(elm) {
		if (this.selected == this) {
			return;
		}
		if (!this.tabs.has(elm)) {
			return;
		}
		sel = this.selected;
		$(sel).removeClass("selected");
		this.tabs.get(sel).hide();
		this.selected = elm;
		$(elm).addClass("selected");
		body = this.tabs.get(elm);
		body.show();
	};

	$(elms).each(function(i, elm) {
		forelm = $(elm).data("for");
		body = $("#"+forelm);
		if (!body) {
			return;
		}
		this.pages[forelm] = {
			header: elm,
			body: body
		};
		this.tabs.set(elm, body);
		if (this.tabs.size==1) {
			$(elm).addClass("selected");
			body.show();
			this.selected = elm;
		} else {
			$(elm).removeClass("selected");
			body.hide();
		}
		$(elm).on("click", {tabs:this}, function(e) {
			e.preventDefault();
			//log($(this).data("for")+" header on: click");
			e.data.tabs.select(this);
		});
	}.bind(this));
}
