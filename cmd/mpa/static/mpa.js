// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

function setupViewMode(params) {
	var p = params;
	var view = document.getElementById("view");
	var nav = document.getElementById("nav");
	var text = document.getElementById("text");
	var n = parseInt(window.location.hash.substr(1));
	for (var i = 0; i < p.photos.length; i++) {
		if (p.photos[i] == n) {
			p.idx = i;
			break;
		}
	}
	view.src = "/image/" + p.photos[p.idx];
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
				view.src = "/image/" + p.photos[p.idx];
				updateNav();
			}
		} else if ((event.clientX - b.left) < b.width/3) {
			if (p.idx > 0) {
				p.idx--;
				view.src = "/image/" + p.photos[p.idx];
				updateNav();
			}
		} else {
			if (requestFullScreen()) {
			} else if (hidden) {
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

function setupDropImage(clickMsg, noSubmitMsg, connectionError) {
	var images = document.getElementById("images");
	var multi = document.getElementById("multi");
	var modal1 = document.getElementById('modal_1');
	var modalErr = document.getElementById('modal_err');
	var error = document.getElementById("error");
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
	function showError(msg) {
		error.firstChild.nodeValue = msg;
		modalErr.checked = true;
	}
	var obj = this;
	this.submit = function() {
		var meta = {name: document.getElementById("albumName").value, descriptions: {}};
		var d = new FormData();
		var ok = false;
		for (var i = 0; i < this.images.length; i++) {
			var o = this.images[i];
			if (o == null) {
				continue;
			}
			d.append("image:" + i, o.file);
			meta.descriptions[i] = o.description;
			ok = true;
		}
		d.append("metadata", JSON.stringify(meta));
		if (meta.name == "" || !ok) {
			showError(noSubmitMsg);
			return;
		}
		prog.show();
		var r = new XMLHttpRequest();
		r.open("POST", "/api/new/album");
		r.onerror = function() {
			showError(connectionError);
		};
		r.upload.addEventListener("progress", function(e) {
			if (e.lengthComputable) {
				prog.update(e.loaded, e.total);
			}
		});
		r.onload = function() {
			if (r.status == 200) {
				document.getElementById("result").innerHTML = r.response;
				images.className = "hidden";
			} else if (r.status == 401) {
				document.getElementById("login").innerHTML = r.response;
				document.getElementById("modal_login").checked = true;
				document.getElementById("login_submit").onclick = function() { loginOnClick(function() { obj.submit(); }); };
			} else {
				showError(r.response);
			}
		};
		r.send(d);
	};
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

function loginOnClick(callback) {
	var login = document.getElementById("login");
	var loginName = document.getElementById("login_name");
	var password = document.getElementById("password");
	var loginMsg = document.getElementById("login_msg");
	var r = new XMLHttpRequest();
	var loginError = document.getElementById("login_error");
	var errorVisible = false;
	function showError(msg) {
		if (!errorVisible) {
			document.getElementById("login_stack").appendChild(document.getElementById("login_error"));
			errorVisible = true;
		}
		loginMsg.firstChild.nodeValue = msg;
		document.getElementById("modal_login").checked = true;
	}
	r.open("POST", "/api/login");
		r.onerror = function() {
			showError(document.getElementById("connection_error").firstChild.nodeValue);
		};
	r.onload = function() {
		if (r.status == 200) {
			callback();
		} else {
			loginName.value = "";
			password.value = "";
			loginName.focus();
			showError(r.response);
		}
	};
	var data = new FormData();
	data.append("login", loginName.value);
	data.append("password", password.value);
	r.send(data);
	return false;
}

function requestFullScreen() {
	var d = document.documentElement;
	if (d.mozRequestFullScreen && !document.mozFullScreenElement) {
		d.mozRequestFullScreen();
		return true;
	} else if (d.webkitRequestFullScreen && !document.webkitFullscreenElement) {
		d.webkitRequestFullScreen();
		return true;
	} else if (d.msRequestFullscreen && !document.msFullscreenElement) {
		d.msRequestFullscreen();
		return true;
	}
	return false;
}
