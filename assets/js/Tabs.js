function Tabs(elms) {
	this.tabs = new Map();
	this.selected = null;

	$(elms).each(function(i, elm) {
		body = $("#"+$(elm).data("for"));
		if (!body) {
			return;
		}
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
			if (e.data.tabs.selected == this) {
				//log("allready selected");
				return;
			}
			sel = e.data.tabs.selected;
			$(sel).removeClass("selected");
			e.data.tabs.tabs.get(sel).hide();
			e.data.tabs.selected = this;
			$(this).addClass("selected");
			e.data.tabs.tabs.get(this).show();
		});
	}.bind(this));
}
