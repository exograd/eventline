"use strict";

function evShowNotification(type, message) {
  let typeClass;

  switch (type) {
  case "info":
    typeClass = "is-info";
    break;
  case "warning":
    typeClass = "is-warning";
    break;
  case "error":
    typeClass = "is-danger";
    break;
  default:
    throw new Error(`invalid notification type '${type}'`);
  }

  let element = document.createElement("div");
  element.appendChild(document.createTextNode(evSentence(message)));
  element.classList.add("notification", typeClass);

  const parent = document.getElementById("ev-notifications");

  const children = parent.querySelectorAll(".notification");
  if (children.length >= 3) {
    parent.removeChild(children[0]);
  }

  parent.appendChild(element);

  if (type != "info") {
    window.scroll(0, 0);
  }
}

function evShowInfo(message) {
  evShowNotification("info", message);
}

function evShowWarning(message) {
  evShowNotification("warning", message);
}

function evShowError(message) {
  evShowNotification("error", message);
}

function evClearNotifications() {
  const parent = document.getElementById("ev-notifications");
  parent.innerHTML = "";
}
