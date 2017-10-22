//require jQuery
function Tabs(elms) {
	this.tabs = new Map();
	this.pages = {};
	this.selected = null;

	this.select = function(elm) {
		if (this.selected == elm) {
			return;
		}
		if (!this.tabs.has(elm)) {
			return;
		}

		if (this.selected !== null) {
			var sel = this.selected;
			$(sel).removeClass("selected");
			this.tabs.get(sel).hide();
		}

		this.selected = elm;
		$(elm).addClass("selected");
		var body = this.tabs.get(elm);
		body.show();
		$(elm).trigger("select");
		if (typeof(userConfig) == "object" && typeof(userConfig.update) == "function") {
			log("tabs: updating active tab in userConfig");
			userConfig.update({
				activeTab: $(elm).data("for")
			});
		}
	};

	var showFirst = null;
	var selFor = "";
	if (typeof(userConfig) == "object" && typeof(userConfig.data) == "function") {
		var data = userConfig.data();
		if (typeof(data.activeTab) == "string") {
			selFor = data.activeTab;
			log("tabs: got active tab from userConfig");
		}
	}
	$(elms).each(function(i, elm) {
		forelm = $(elm).data("for");
		body = $("#"+forelm);
		if (!body) {
			return;
		}
		$(elm).removeClass("selected");
		body.hide();

		this.pages[forelm] = {
			header: elm,
			body: body
		};
		this.tabs.set(elm, body);

		if (this.tabs.size==1) {
			showFirst = elm;
		}
		if (selFor != "" && forelm === selFor) {
			showFirst = elm;
		}

		$(elm).on("click", {tabs:this}, function(e) {
			e.preventDefault();
			log($(this).data("for")+" header on: click");
			e.data.tabs.select(this);
			log($(this).data("for")+" header on: click end");
		});
	}.bind(this));
	if (showFirst !== null)  {
		this.select(showFirst);
	}
}
