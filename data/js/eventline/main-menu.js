"use strict";

function evSetupMainMenu() {
  const expandLink = evMainMenuExpandLink();
  expandLink.onclick = (event) => {
    event.preventDefault();

    evExpandMainMenu();
    window.localStorage.setItem("mainMenuState", "expanded");
  };

  const collapseLink = evMainMenuCollapseLink();
  collapseLink.onclick = (event) => {
    event.preventDefault();

    evCollapseMainMenu();
    window.localStorage.setItem("mainMenuState", "collapsed");
  };

  window.onresize = (event) => {
    evAutoExpandOrCollapseMainMenu();
  };

  const mainMenuState = window.localStorage.getItem("mainMenuState");
  switch (mainMenuState) {
  case "expanded":
    evExpandMainMenu();
    break;

  case "collapsed":
    evCollapseMainMenu();
    break;

  default:
    evAutoExpandOrCollapseMainMenu();
    break;
  }
}

function evAutoExpandOrCollapseMainMenu() {
  const mainMenuState = window.localStorage.getItem("mainMenuState");
  if (mainMenuState) {
    return;
  }

  if (window.innerWidth < 1408) { // Bulma's fullhd
    evCollapseMainMenu();
  } else {
    evExpandMainMenu();
  }
}

function evExpandMainMenu() {
  const expandLink = evMainMenuExpandLink();
  const collapseLink = evMainMenuCollapseLink();

  expandLink.classList.add("is-hidden");
  collapseLink.classList.remove("is-hidden");

  const menu = evMainMenu();
  menu.classList.add("ev-expanded");

  const content = evContent();
  content.classList.add("ev-menu-expanded");
}

function evCollapseMainMenu() {
  const expandLink = evMainMenuExpandLink();
  const collapseLink = evMainMenuCollapseLink();

  collapseLink.classList.add("is-hidden");
  expandLink.classList.remove("is-hidden");

  const menu = evMainMenu();
  menu.classList.remove("ev-expanded");

  const content = evContent();
  content.classList.remove("ev-menu-expanded");
}

function evMainMenuExpandLink() {
  return document.getElementById("ev-expand-menu");
}

function evMainMenuCollapseLink() {
  return document.getElementById("ev-collapse-menu");
}

function evMainMenu() {
  return document.getElementById("ev-menu");
}

function evContent() {
  return document.getElementById("ev-content");
}
