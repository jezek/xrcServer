//require jQuery
$(function() {
	$("#pair_form").on("submit", function(e){
		e.preventDefault();
		var passphrase = $("input[name='passphrase']").val();
		if (!passphrase) {
			return;
		}
		
		var url=window.location.href+passphrase;
		window.location=url;
		
	});
});
