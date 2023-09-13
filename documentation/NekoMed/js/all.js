

function ValidateIPaddress(ipaddress) {  
  if (/^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/.test(ipaddress)) {  
    return (true)  
  }  
  //alert("You have entered an invalid IP address!")  
  return (false)  
}  

jQuery(document).ready(function() {
    console.log("loading...");
    jQuery('#exampleAE').on('change', function() {
	var t = jQuery('#exampleAE').val();
	jQuery('#exampleAE').val(t.replace(/[^a-zA-z0-9]/g,""));
    });

    jQuery('#examplePort').on('change', function() {
	var p = parseInt(jQuery('#examplePort').val());
	if (p < 1024) {
	    alert("Warning: your port number is smaller than 1024. Ports in that range may be restricted on your network.");
	}
    });
    
    jQuery("#submitConnection").on('click', function(event) {
	event.preventDefault();
	var IP = jQuery('#exampleIP').val();
	var AETitle = jQuery('#exampleAE').val();
	AETitle = AETitle.trim();
	var Port = parseInt(jQuery('#examplePort').val());
	if (!ValidateIPaddress(IP)) {
	    alert("The IP number you entered does not match an official IP. Try again.");
	    return;
	}
	if (AETitle == "") {
	    alert("Error: provide an application entity title for your service.");
	    return;
	}
	if (AETitle.length < 3 || AETitle.length > 10) {
	    alert("Error: try to provide an AETitle with a length between 3 and 15 characters.");
	    return;
	}
	if (Port < 1) {
	    alert("Error: specify a port value in the range of 1 to 65,534.");
	    return;
	}
	// now save this connection
	jQuery.getJSON('php/saveConnection.php', { "IP": IP, "AETitle": AETitle, "Port": Port }, function(msg) {
	    if (typeof msg["message"] != 'undefined') {
		alert(msg.message);
		return;
	    } 
	    // ok, worked, so remove the entries now
	    console.log("added a listener");
	    jQuery('#exampleAE').val("");
	    jQuery('#exampleIP').val("");
	    jQuery('#examplePort').val("");
	    alert("Done, check your scores");
	});
	
    });

    setInterval(function() {
	// update the leader board
	jQuery.getJSON('php/connections.json', function(data) {
	    // compute the sum of points
	    data.map(function(a) {
		a.Points = 3 + parseInt(a.SCUWorking) + (0.1*parseInt(a.SeriesReceived)) + (10000 * a.ReceivedStudy.length);
	    });	    
	    data.sort(function(a,b) {
		if (a.Points < b.Points)
		    return -1;
		if (a.Points == b.Points)
		    return 0;
		return 1;
	    });
	    jQuery('#leaderboard').children().remove();
	    var counter = 1;
	    for (var i = 0; i < data.length; i++) {
		if (typeof data[i].Points != 'undefined' && data[i].Points > 0) {
		    jQuery('#leaderboard').append("<div class='entry'><div class='place'>"+counter+"</div><div class='team'>Team " + data[i].AETitle + "</div><div class='Points'>" + data[i].Points.toFixed(2) + " points</div></div>");
		    counter = counter + 1;
		}
	    }
	    if (counter == 1) {
		jQuery('#leaderboard').append("<span class='text-white-50'>No team has any points yet.</span>");
	    }
	});
    }, 1000);
    
    // update the list of connections
    setInterval(function() {
	jQuery.getJSON('php/connections.json', function(data) {
	    data.sort(function (a, b) {
		if (a.IP < b.IP)
		    return -1;
		if (a.IP == b.IP)
		    return 0;
		return 1;
	    });
	    jQuery('#connections').children().remove();
	    for (var i = 0; i < data.length; i++) {
		var active = (data[i].SCUWorking=="1")?"working":"broken";
		var icon = "fa-thumbs-up";
		if (active == "broken") {
		    icon = "fa-thumbs-down";
		}
		jQuery('#connections').append("<div class='connection "+active+"'><div class='IP' title='IP number'>"+data[i].IP+"</div><div class='AETitle' title='Application Entity Title'>"+data[i].AETitle+"</div><div class='Port' title='Port number'>"+data[i].Port+"</div><div class='ok'><i class='fa-solid " + icon + "'></i></div><div class='numSeries' title='Received series'>("+data[i].SeriesReceived+")</div></div>");

		// add as an option to exampleConnection
		var options = jQuery('#exampleConnection').children();
		var alreadyThere = false;
		var k = data[i].IP + data[i].AETitle + data[i].Port;
		for (var j = 0; j < options.length; j++) {
		    var val = jQuery(options[j]).attr('value');
		    if (val == k)
			alreadyThere = true;
		}
		if (!alreadyThere) {
		    jQuery('#exampleConnection').append('<option value="' + k + '">'+ data[i].IP + " " + data[i].AETitle + " " + data[i].Port +'</option>');
		}
	    }
	});
    }, 1000);

    jQuery('#submitStudy').on('click', function(event) {
	event.preventDefault();
	var entry = jQuery('#exampleConnection option:selected').attr('value');
	var PatientID = jQuery('#examplePatientID').val().trim();
	var StudyInstanceUID = jQuery('#exampleStudyInstanceUID').val().trim();
	var SeriesInstanceUID = jQuery('#exampleSeriesInstanceUID').val().trim();
	if (PatientID == "" || StudyInstanceUID == "" || SeriesInstanceUID == "" || entry == "") {
	    alert("Error: we need entries for all three to check for a series");
	    return;
	}
	
	jQuery.getJSON('php/reportStudy.php', { "entry": entry, "PatientID": PatientID, "StudyInstanceUID": StudyInstanceUID, "SeriesInstanceUID": SeriesInstanceUID }, function(data) {
	    if (typeof data["message"] != 'undefined') {
		alert(data["message"]);
		return;
	    }
	    // reset the form if this worked
	    jQuery('#examplePatientID').val("");
	    jQuery('#exampleStudyInstanceUID').val("");
	    jQuery('#exampleSeriesInstanceUID').val("");
	    alert("Done! Check your scores.");
	    return;
	});
    });

    hljs.highlightAll();
});
