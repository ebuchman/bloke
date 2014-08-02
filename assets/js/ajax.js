
// general framework for ajax calls. _callback functions are in callback.js

function new_request_obj(){
    if (window.XMLHttpRequest)
        return new XMLHttpRequest();
    else
        return new ActiveXObject("Microsoft.XMLHTTP");
}

function request_callback(xmlhttp, _func, args){
    xmlhttp.onreadystatechange=function(){
        if (xmlhttp.readyState==4 && xmlhttp.status==200){
		args.unshift(xmlhttp);
		_func.apply(this, args);
        }
    }
}

function make_request(xmlhttp, method, path, async, params){
    xmlhttp.open(method, path, async);
    xmlhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
    var s = "", i = 0;
    for (var k in params){
	if (i > 0)
  	  s += "&";
	s += k+"="+encodeURIComponent(params[k]);
	i++;
    }
    //xmlhttp.setRequestHeader("Content-length", s.length); // important?
    xmlhttp.send(s);
}


// inplace is either nothing or is the element corresponding to the bubble to change
function get_entry_data(name, inplace){
    inplace = typeof inplace !== 'undefined' ? inplace : 0

    xmlhttp = new_request_obj();
    request_callback(xmlhttp, _get_entry_data, [name, inplace]);
    make_request(xmlhttp, "POST", "index.php", true, {"form":"load_content", "name":name});
    return false;
}

function displaySearchResults(keystrokes){
    if (keystrokes.length == 0)
    { document.getElementById('search_results').innerHTML = ""; return;}

    xmlhttp = new_request_obj();

    // make callback even?
    xmlhttp.onreadystatechange=function(){
        if (xmlhttp.readyState==4 && xmlhttp.status==200){
    	    document.getElementById('search_results').innerHTML = xmlhttp.responseText;
        }
    }
    make_request(xmlhttp, "POST", "index.php", true, {"form":"search_db", "keystrokes":keystrokes});
    return false;
}
