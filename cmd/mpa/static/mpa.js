// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

function setupViewMode(params) {
	var p = params;
	var view = document.getElementById("view");
	var nav = document.getElementById("nav");
	var text = document.getElementById("text");
	function updateNav() {
		text.firstChild.nodeValue = "" + (p.idx + 1) + " / " + p.photos.length;
	}
	updateNav();
	var hidden = true;
	view.addEventListener("click", function(event) {
		var b = view.getBoundingClientRect();
		if ((event.clientX - b.left) > 2*b.width/3) {
			if (p.idx < p.photos.length - 1) {
				p.idx++;
				view.src = p.photos[p.idx];
				updateNav();
			}
		} else if ((event.clientX - b.left) < b.width/3) {
			if (p.idx > 0) {
				p.idx--;
				view.src = p.photos[p.idx];
				updateNav();
			}
		} else {
			if (hidden) {
				nav.className = "";
				hidden = false;
			} else {
				nav.className = "hidden";
				hidden = true;
			}
		}
	}, false);
}
