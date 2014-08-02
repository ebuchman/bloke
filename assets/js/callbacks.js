
function _get_entry_data(xmlhttp, name, inplace){
	var response = JSON.parse(xmlhttp.responseText);
	var content = response.content;

	var owner = response.owner;
	if (inplace == 0)
		new_bubble(name, content, owner);
	else 
		reload_bubble(name, content, inplace)
}

