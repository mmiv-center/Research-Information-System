<?php

  $IP = "";
  $AETitle = "";
  $Port = "";
  if (isset($_GET['IP'])) {
     $IP = $_GET['IP'];
  } else {
     echo(json_encode(array("message" => "Error: no IP address provided.")));
     exit(-1);
  }

  if (isset($_GET['AETitle'])) {
     $AETitle = $_GET['AETitle'];
  } else {
     echo(json_encode(array("message" => "Error: no AETitle provided.")));
     exit(-1);
  }

  if (isset($_GET['Port'])) {
     $Port = $_GET['Port'];
  } else {
     echo(json_encode(array("message" => "Error: no Port provided.")));
     exit(-1);
  }

  $db_file = "connections.json";

  $data = json_decode(file_get_contents($db_file), TRUE);
  $found = False;
  foreach ($data as $dat) {
     if ($dat["IP"] == $IP && $dat["Port"] == $Port) {
         $found = True;
     }
  }
  if (!$found) {
     // add and save
     $data[] = array("IP" => $IP, "AETitle" => $AETitle, "Port" => $Port, "SCUWorking" => 0, "SeriesReceived" => 0, "ReceivedStudy" => array());
     file_put_contents($db_file, json_encode($data));
     exit(0); // done without an error     
  } else {
    echo(json_encode(array( "message" => "Error: A connection to this system exists already. IP and port number need to be unique." )));
    exit(0);
  }

?>