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
		log("tabs: triggering select");
		$(elm).trigger("select");
		if (typeof(userConfig) == "object" && typeof(userConfig.update) == "function") {
			log("tabs: updating active tab in userConfig: "+$(elm).data("for"));
			userConfig.update({
				activeTab: $(elm).data("for")
			});
		}
	};

	var selFor = "";
	if (typeof(userConfig) == "object" && typeof(userConfig.data) == "function") {
		var data = userConfig.data();
		if (typeof(data.activeTab) == "string") {
			selFor = data.activeTab;
			log("tabs: got active tab from userConfig: "+selFor);
		}
	}
	var found=false;
	var first=null;
	$(elms).each(function(i, elm) {
		forelm = $(elm).data("for");
		body = $("#"+forelm);
		if (!body) {
			return;
		}
		if (i==0) {
			first = {
				elm: elm,
				body: body
			};
		}
		if (found===false && (forelm === selFor || (selFor=="" && i==0))) {
			$(elm).addClass("selected");
			this.selected=elm;
			found=true;
		  body.show();
		} else {
		  body.hide();
			$(elm).removeClass("selected");
		}

		this.pages[forelm] = {
			header: elm,
			body: body
		};
		this.tabs.set(elm, body);

		$(elm).on("click", {tabs:this}, function(e) {
			e.preventDefault();
			log($(this).data("for")+" header on: click");
			e.data.tabs.select(this);
			log($(this).data("for")+" header on: click end");
		});
	}.bind(this));

	if (!found && first) {
		$(first.elm).addClass("selected");
		this.selected=first.elm;
		first.body.show();
	}

}
