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

function progress() {
	var prog = document.getElementById('progress');
	var percent = document.getElementById('percent');
	this.show = function() { prog.className = "progress"; };
	this.update = function(part, total) {
		percent.style.width = (100 * part / total) + "%";
	};
}

function setupDropImage(clickMsg) {
	var images = document.getElementById("images");
	var multi = document.getElementById("multi");
	var modal1 = document.getElementById('modal_1');
	var description = document.getElementById('description');
	var prog = new progress();
	this.images = [];
	this.modalIdx = 0;
	this.addDescription = function() {
		var o = this.images[this.modalIdx];
		var old = o.description;
		o.description = description.value;
		if (description.value == "") {
			if (old != "") {
				o.span.className = "hidden";
			}
			return;
		}
		o.span.firstChild.nodeValue = description.value;
		if (old == "") {
			o.span.className = "label success full";
		}
	};
	this.deleteImage = function() {
		var o = this.images[this.modalIdx];
		images.removeChild(o.div);
		this.images[this.modalIdx] = null;
	};
	this.showModal = function(idx) {
		this.modalIdx = idx;
		description.value = this.images[idx].description;
		modal1.checked = true; 
		return false;
	};
	this.submit = function() {
		var d = new FormData();
		for (var i = 0; i < this.images.length; i++) {
			var o = this.images[i];
			if (o == null) {
				continue;
			}
			d.append("image", o.file);
		}
		prog.show();
		var r = new XMLHttpRequest();
		r.open("POST", "/new");
		r.onerror = function() {
			console.log("Connection error");
		};
		r.upload.addEventListener("progress", function(e) {
			if (e.lengthComputable) {
				prog.update(e.loaded, e.total);
			}
		});
		r.onload = function() {
			console.log("onload: " + r.status);
		};
		r.send(d);
	};
	var obj = this;
	this.addImage = function(file) {
		var input = document.createElement("input");
		input.setAttribute("title", clickMsg);
		input.setAttribute("type", "file");
		var idx = this.images.length;
		input.onclick = function () { return obj.showModal(idx); };
		var label = document.createElement("label");
		label.appendChild(input);
		label.className = "dropimage";
		var span = document.createElement("span");
		span.className = "hidden";
		span.appendChild(document.createTextNode(description.value));
		var div = document.createElement("div");
		div.appendChild(label);
		div.appendChild(span);
		if (URL.createObjectURL) {
			label.style['background-image'] = 'url('+URL.createObjectURL(file)+')';
		} else {
			var reader = new FileReader();
			reader.onloadend = function(){
				label.style['background-image'] = 'url('+reader.result+')';
			};
			reader.readAsDataURL(file);
		}
		images.insertBefore(div, multi);
		this.images.push({div: div, file: file, span: span, description: ""});
	};
	document.querySelector('.dropimage').onchange = function(e){
		for (var i = 0; i < e.target.files.length; i++) {
			obj.addImage(e.target.files[i]);
		}
	};
}
