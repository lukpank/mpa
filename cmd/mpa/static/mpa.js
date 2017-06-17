// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

function setupViewMode(params) {
	var p = params;
	var nav = document.getElementById("nav");
	var text = document.getElementById("text");
	var body = document.body;
	var n = parseInt(window.location.hash.substr(1));
	for (var i = 0; i < p.images.length; i++) {
		if (p.images[i] == n) {
			p.idx = i;
			break;
		}
	}
	function updateNav() {
		text.firstChild.nodeValue = "" + (p.idx + 1) + " / " + p.images.length;
	}
	var next = new Image();
	function handleError(idx) {
		var r = new XMLHttpRequest();
		r.open("GET", "/api/image/" + p.images[idx]);
		setupHTTPEventListeners(r, p.connectionError, function() { showImage(idx); }, null);
		r.send();
	}
	var timeout = null;
	function showImage(idx, slideShow) {
		if (!slideShow) {
			clearTimeout(timeout);
			timeout = null;
		}
		if (idx < 0 || idx >= p.images.length) {
			return;
		}
		var src = "/image/" + p.images[idx];
		next.onerror = function() { handleError(idx); };
		next.onload = function() {
			p.idx = idx;
			p.width = next.width;
			p.height = next.height;
			body.style.backgroundImage = "url(" + src + ")";
			updateNav();
			if (slideShow) {
				timeout = setTimeout(showImage, 3000, idx + 1, true);
			}
		};
		next.src = src;
	}
	showImage(p.idx);
	document.onkeydown = function(e) {
		if (e.keyCode == 32) {
			showImage(p.idx + 1, false);
		} else if (e.keyCode == 8) {
			showImage(p.idx - 1, false);
		}
	};
	p.slideShow = function () {
		nav.className = "hidden";
		hidden = true;
		timeout = setTimeout(showImage, 3000, p.idx + 1, true);
	};
 	var hidden = true;
	body.addEventListener("click", function(event) {
		if (event.target != body) {
			return;
		}
		var b = body.getBoundingClientRect();
		var w = p.width / p.height * b.height;
		if (w > b.width) {
			w = b.width;
		}
		var left = (b.width - w) / 2;
		if ((event.clientX - left) > 2*w/3) {
			showImage(p.idx + 1, false);
		} else if ((event.clientX - left) < w/3) {
			showImage(p.idx - 1, false);
		} else {
			clearTimeout(timeout);
			timeout = null;
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
	var albumName = document.getElementById('albumName');
	this.show = function() {
		prog.className = "progress";
		albumName.className = "hidden"; 
	};
	this.hide = function() {
		prog.className = "hidden";
		percent.style.width = "0%";
		albumName.className = "";
	};
	this.update = function(part, total) {
		percent.style.width = (100 * part / total) + "%";
	};
}

function showError(msg) {
	var err = document.getElementById("err")
	var modal = document.getElementById('modal_err');
	document.getElementById("error").firstChild.nodeValue = msg;
	err.onkeydown = function(e) {
		if (e.keyCode == 13 || e.keyCode == 27) {
			modal.checked = false;
			err.blur();
		}
	};
	modal.checked = true;
	err.focus();
}

function setupHTTPEventListeners(r, connectionError, callback, onResponse) {
	r.onerror = function() {
		showError(connectionError);
		if (onResponse != null) {
			onResponse(null);
		}
	};
	r.onload = function() {
		if (r.status == 200) {
		} else if (r.status == 401) {
			var login = document.getElementById("login");
			login.onkeydown = function(e) {
				if (e.keyCode == 13) {
					document.getElementById("modal_login").checked = false;
					loginOnClick(function() { callback(); });
					return false;
				} else if (e.keyCode == 27) {
					document.getElementById("modal_login").checked = false;
				}
			};
			login.innerHTML = r.response;
			document.getElementById("modal_login").checked = true;
			document.getElementById("login_submit").onclick = function() { loginOnClick(function() { callback(); }); };
		} else {
			showError(r.response);
		}
		if (onResponse != null) {
			onResponse(r.status);
		}
	};
}

function setupEditAlbum(submitURL, origName, imgs, clickMsg, noSubmitMsg, connectionError) {
	var images = document.getElementById("images");
	var multi = document.getElementById("multi");
	var modal1 = document.getElementById('modal_1');
	var title = document.getElementById('title');
	var upload = document.getElementById("upload");
	var prog = new progress();
	for (var i = 0; i < imgs.length; i++) {
		imgs[i].origTitle = imgs[i].title;
	}
	this.isEdit = imgs.length > 0;
	this.origName = origName;
	this.images = imgs;
	this.deleted = [];
	this.modalIdx = 0;
	this.addTitle = function() {
		var o = this.images[this.modalIdx];
		var span = document.getElementById("title_"+this.modalIdx);
		var old = o.title;
		o.title = title.value;
		if (title.value == "") {
			if (old != "") {
				span.className = "hidden";
			}
			return;
		}
		span.firstChild.nodeValue = title.value;
		if (old == "") {
			span.className = "label success full";
		}
	};
	this.deleteImage = function() {
		var idx = this.modalIdx;
		var o = this.images[idx];
		var div = document.getElementById("img_"+idx);
		images.removeChild(div);
		if (o.id != null) {
			this.deleted.push(o.id);
		}
		this.images[idx] = null;
	};
	this.edit = function(idx) {
		this.modalIdx = idx;
		title.value = this.images[idx].title;
		title.focus();
		modal1.checked = true; 
		return false;
	};
	var obj = this;
	document.getElementById("edit").onkeydown = function(e) {
		if (e.keyCode == 13) {
			obj.addTitle();
			modal1.checked = false;
		} else if (e.keyCode == 27) {
			modal1.checked = false;
		}
	};
	this.submit = function() {
		var meta = {name: document.getElementById("albumName").value, titles: {}, edit: {deleted: this.deleted, titles: {}}};
		var d = new FormData();
		var ok = this.deleted.length > 0 || (this.isEdit && meta.name != this.origName);
		for (var i = 0; i < this.images.length; i++) {
			var o = this.images[i];
			if (o == null) {
				continue;
			}
			if (o.id != null) {
				if (o.title != o.origTitle) {
					meta.edit.titles[o.id] = o.title;
					ok = true;
				}
			} else {
				d.append("image:" + i, o.file);
				meta.titles[i] = o.title;
				ok = true;
			}
		}
		d.append("metadata", JSON.stringify(meta));
		if (meta.name == "" || !ok) {
			showError(noSubmitMsg);
			return;
		}
		upload.disabled = true;
		prog.show();
		var r = new XMLHttpRequest();
		r.open("POST", submitURL);
		setupHTTPEventListeners(
			r, connectionError, function() { obj.submit(); },
			function(status) {
				if (status == 200) {
					document.getElementById("result").innerHTML = r.response;
					images.className = "hidden";
					document.getElementById("bmenu").checked = false;
				} else {
					upload.disabled = false;
					prog.hide();
				}
			});
		r.upload.addEventListener("progress", function(e) {
			if (e.lengthComputable) {
				prog.update(e.loaded, e.total);
			}
		});
		r.send(d);
	};
	this.addImage = function(file) {
		var input = document.createElement("input");
		input.setAttribute("title", clickMsg);
		input.setAttribute("type", "file");
		var idx = this.images.length;
		input.onclick = function () { return obj.edit(idx); };
		var label = document.createElement("label");
		label.appendChild(input);
		label.className = "dropimage";
		var span = document.createElement("span");
		span.id = "title_" + idx;
		span.className = "hidden";
		span.appendChild(document.createTextNode(title.value));
		var div = document.createElement("div");
		div.id = "img_" + idx;
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
		this.images.push({id: null, file: file, title: ""});
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
