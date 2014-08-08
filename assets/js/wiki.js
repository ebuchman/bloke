
// general framework for ajax calls. 

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
    inplace = typeof inplace !== 'undefined' ? inplace : 0;

    console.log(document.documentElement.scrollTop);
    console.log(document.body.scrollTop);

    xmlhttp = new_request_obj();
    request_callback(xmlhttp, _get_entry_data, [name, inplace, document.documentElement.scrollTop]);
    make_request(xmlhttp, "POST", "/bubbles/"+name+".md", true, {"form":"load_content", "name":name});
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

// bubble functions

function new_bubble(name, content, pos){
    var workflow_div = document.getElementById('workflow');
    var proto = document.getElementById('entry_div_box_proto').cloneNode(true);

    proto.getElementsByClassName('entry_content_box')[0].innerHTML = content;
	
//    MathJax.Hub.Typeset(proto);

    proto.getElementsByClassName('content_header')[0].innerHTML = "<h4>"+name+"</h4>";
    proto.setAttribute("class", "entry_div_box content_unit");
    proto.setAttribute("id", "entry_div_box_"+name);
    //proto.setAttribute("style", "position:absolute;margin-top:"+pos.toString()+"px;");

    workflow_div.insertBefore(proto, workflow_div.children[1]);
}

function reload_bubble(name, content, element){
    element.getElementsByClassName('entry_content_box')[0].innerHTML = content;
    element.getElementsByClassName('content_header')[0].innerHTML = "<h4>"+name+"</h4>";
    MathJax.Hub.Queue(["Typeset",MathJax.Hub]);
}

function close_bubble(id){
 document.getElementById(id).remove();
}

// callback

function _get_entry_data(xmlhttp, name, inplace, pos){
	var content = xmlhttp.responseText; //JSON.parse(xmlhttp.responseText);
    new_bubble(name, content, pos);
}



