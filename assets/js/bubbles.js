function new_bubble(name, content, owner){
    var workflow_div = document.getElementById('workflow');
    var proto = document.getElementById('entry_div_box_proto').cloneNode(true);

    proto.getElementsByClassName('entry_content_box')[0].innerHTML = content;
	
    MathJax.Hub.Typeset(proto);

    if (owner){
        // edit and delete links, built dynamically
        var delete_link = proto.getElementsByClassName('delete_link')[0];
        var edit_link = proto.getElementsByClassName('edit_link')[0];

        delete_link.href="javascript:;"; 
        delete_link.onclick = function(){delete_entry(name);};
        delete_link.innerHTML="delete";

        edit_link.href="#/";
        edit_link.onclick= function(){edit_entry_data(name, content, proto);};
        edit_link.innerHTML="edit";
    }
    proto.getElementsByClassName('content_header')[0].innerHTML = "<h4>"+name+"</h4>";
    proto.setAttribute("class", "entry_div_box content_unit");
    proto.setAttribute("id", "entry_div_box_"+name);

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


