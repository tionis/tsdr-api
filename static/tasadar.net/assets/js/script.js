var trelloToken, trelloListID, greetMsg

function date() {
  let currentDate = new Date();
  let dateOptions = {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric"
  };
  let date = currentDate.toLocaleDateString("en-GB", dateOptions);
  var currentTime = new Date();
  var currentHours = currentTime.getHours();
  var currentMinutes = currentTime.getMinutes();
  var currentSeconds = currentTime.getSeconds();
  currentHours = (currentHours < 10 ? "0" : "") + currentHours;
  currentMinutes = (currentMinutes < 10 ? "0" : "") + currentMinutes;
  currentSeconds = (currentSeconds < 10 ? "0" : "") + currentSeconds;
  var time = currentHours + ":" + currentMinutes + "<span style=\"color:grey\">:" + currentSeconds + "</span>";
  document.getElementById("header_date").innerHTML = date + " - " + time;
}

function greet() {
  document.getElementById("header_greet").innerHTML = getGreeting()
}

function getGreeting() {
  let greeting;
  let currentTime = new Date();
  let greet = Math.floor(currentTime.getHours() / 6);
  switch (greet) {
    case 0:
      greeting = "Good night!";
      break;
    case 1:
      greeting = "Good morning!";
      break;
    case 2:
      greeting = "Good afternoon!";
      break;
    case 3:
      greeting = "Good evening!";
      break;
  }
  return greeting;
}

function checkForUpdate() {
  if (trelloToken) {
    if (trelloListID) {
      getList();
    } else {
      document.getElementById("header_info").innerHTML = '<h2 id="header_info">No list selected!</h2>';
    }
  } else {
    document.getElementById("header_info").innerHTML = '<h2 id="header_info">No trello token given!</h2>';
  }
}

function handleTrelloResponse(response) {
  if (response.ok) {
    return response.json();
  } else if (response.status == 401) {
    throw Error("Token invalid")
  } else {
    console.log(response.status)
    throw Error("Unknown error received")
  }
}

async function getList() {
  fetch('https://api.trello.com/1/lists/' + trelloListID + '/cards?key=99c874600c9b205bf4804ac518787b71&token=' + trelloToken)
    .then(handleTrelloResponse)
    .then(function (data) {
      if (data.length > 0) {
        document.title = "(!) Dashboard";
        document.getElementById("header_info").innerHTML = '<h2 id="header_info">You have Cards in your Inbox!</h2>';
      } else {
        document.title = "Dashboard"
        document.getElementById("header_info").innerHTML = '';
      }
    })
    .catch(function (error) {
      if (error == "Token invalid") {
        document.getElementById("header_info").innerHTML = '<h2 id="header_info">Invalid Token!</h2>';
        localStorage.removeItem("trello-token");
        console.log(error);
      } else {
        document.getElementById("header_info").innerHTML = '<h2 id="header_info">Error contacting Trello</h2>';
        console.log(error);
      }
    });
}

function boardChooser() {
  trelloToken = document.getElementById("trello_token_input").value;
  localStorage.setItem("trello-token", trelloToken);
  parseTrelloOptions();
}

function parseTrelloOptions() {
  fetch('https://api.trello.com/1/members/me/boards?fields=name&key=99c874600c9b205bf4804ac518787b71&token=' + trelloToken)
    .then(handleTrelloResponse)
    .then(function (data) {
      let boardSelector = `Select Board: <select id="trello_board_selector" onchange="boardSelected();">`;
      let i;
      for (i = 0; i < data.length; i++) {
        boardSelector += `<option value="` + data[i].id + `">` + data[i].name + `</option>\n`
      }
      boardSelector += `</select>`;
      document.getElementById("trello_board_config").innerHTML = boardSelector;
    })
    .catch(function (error) {
      document.getElementById("trello_board_config").innerHTML = error;
    });
}

function boardSelected() {
  let boardID = document.getElementById("trello_board_selector").value;
  fetch('https://api.trello.com/1/boards/' + boardID + '/lists?key=99c874600c9b205bf4804ac518787b71&token=' + trelloToken)
    .then(handleTrelloResponse)
    .then(function (data) {
      trelloListID = data[0].id
      localStorage.setItem("trello-listid", trelloListID);
      getList();
      let boardSelector = `Select List: <select id="trello_list_selector" onchange="listSelected();">`;
      let i;
      for (i = 0; i < data.length; i++) {
        boardSelector += `<option value="` + data[i].id + `">` + data[i].name + `</option>\n`
      }
      boardSelector += `</select>`;
      document.getElementById("trello_list_config").innerHTML = boardSelector;
    }).catch(function (error) {
      document.getElementById("trello_list_config").innerHTML = error;
    });
}

function listSelected() {
  trelloListID = document.getElementById("trello_list_selector").value;
  localStorage.setItem("trello-listid", trelloListID);
  getList();
}

function checkCompability() {
  if (typeof (Storage) !== "undefined") {
    return true;
  } else {
    document.getElementById("header_greet").innerHTML = "Unsupported Browser!";
    return false;
  }
}

function updateGreet() {
  let newGreet = getGreeting();
  if (greetMsg != newGreet) {
    greetMsg = newGreet;
    document.getElementById("header_greet").innerHTML = getGreeting();
  }
}

function loadConfig() {
  trelloToken = localStorage.getItem("trello-token");
  trelloListID = localStorage.getItem("trello-listid");
}

function loadFunctions() {
  var compatible = checkCompability();
  date();
  var t1 = setInterval(date, 1000);
  if (compatible) {
    loadConfig();
    checkForUpdate();
    var t2 = setInterval(checkForUpdate, 120000);
    greet();
    var t3 = setInterval(updateGreet, 60000)
  }
  if (trelloToken) {
    parseTrelloOptions();
  }
}